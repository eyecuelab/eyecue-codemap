package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"fmt"
	"hash"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"text/template"

	"github.com/akamensky/base58"
	"github.com/mattn/go-isatty"
	"golang.org/x/sync/errgroup"
)

type TokenLocation struct {
	filename   string
	lineNum    int
	linkToFile bool
}

type TokenGroupInfo struct {
	token           string
	fileSource      FileSource
	startLineNumber int
	endLineNumber   int
	actualHash      string
	expectedHash    string
}

type FileInventory struct {
	SinglesByToken      map[string][]TokenLocation
	GroupsByToken       map[string][]TokenGroupInfo
	MarkdownFileSources []FileSource
	sync.Mutex
}

func tagBaseName() string {
	s := os.Getenv("CODEMAP_TAG_BASE")
	if s != "" {
		return s
	}

	return "eyecue-codemap"
}

var tokenNeededRegexp = regexp.MustCompile(fmt.Sprintf(`\[%s(-group)?]`, tagBaseName()))
var tokenRegexp = regexp.MustCompile(fmt.Sprintf(`^(.*)\[%s:([A-Za-z0-9]+)](.*)$`, tagBaseName()))
var tokenGroupStartRegexp = regexp.MustCompile(fmt.Sprintf(`\[%s-group:([A-Za-z0-9]+)]`, tagBaseName()))
var tokenGroupEndRegexp = regexp.MustCompile(fmt.Sprintf(`\[end-%s-group:([A-Za-z0-9]+)(:([a-f0-9]{40}))?]`, tagBaseName()))
var tokenRefRegexp = regexp.MustCompile(fmt.Sprintf(`<!--%s:[A-Za-z0-9]+-->]\(.*?\)`, tagBaseName()))
var tokenGroupRefRegexp = regexp.MustCompile(fmt.Sprintf(`(?s)(<!--%s-group:([A-Za-z0-9]+):(.+?)-->)\n(.*?)(<!--end-%s-group-->)`, tagBaseName(), tagBaseName()))

var ignoreExtensions = []string{
	".csv",
	".jpeg",
	".jpg",
	".otf",
	".png",
	".ttf",
	".webp",
	".woff",
	".woff2",
}

var Version = "dev"

type FilenameSource int

const (
	FilenameSourceStdin FilenameSource = iota
	FilenameSourceStdinNul
	FilenameSourceGit
	FilenameSourceGitIndex
)

type Config struct {
	AckGroups      bool
	CheckOnly      bool
	FilenameSource FilenameSource
	NoUnused       bool
	Verbose        bool
}

type FileSource struct {
	Filename     string
	FromGitIndex bool
}

func main() {
	var config Config

	for _, arg := range os.Args[1:] {
		switch arg {
		case "--help", "-h":
			fmt.Printf("eyecue-codemap version %s\n"+
				"Usage: eyecue-codemap [--check-only] [--git-index] [--no-unused]\n"+
				"If not using --git-index, pipe in a list of filenames to stdin, one per line.\n", Version)
			os.Exit(0)
		case "ack":
			config.AckGroups = true
		case "--check-only":
			config.CheckOnly = true
		case "--git":
			config.FilenameSource = FilenameSourceGit
		case "--git-index":
			config.FilenameSource = FilenameSourceGitIndex
		case "--no-unused":
			config.NoUnused = true
		case "--stdin":
			config.FilenameSource = FilenameSourceStdin
		case "--stdin0":
			config.FilenameSource = FilenameSourceStdinNul
		case "--verbose":
			config.Verbose = true
		default:
			fmt.Printf("ERROR: unrecognized argument: %s\n", arg)
			os.Exit(2)
		}
	}

	if config.FilenameSource == FilenameSourceGitIndex && !config.CheckOnly {
		fmt.Println("ERROR: --check-only must be specified when using --git-index")
		os.Exit(2)
	}

	if config.AckGroups && config.CheckOnly {
		fmt.Println("ERROR: cannot specify both ack and --check-only")
		os.Exit(2)
	}

	err := run(config)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		fmt.Println("eyecue-codemap completed with errors")
		os.Exit(1)
	}

	fmt.Println("eyecue-codemap completed successfully")
}

func run(config Config) error {
	var modeDesc string

	switch config.FilenameSource {
	case FilenameSourceGit:
		modeDesc = "Git"
	case FilenameSourceGitIndex:
		modeDesc = "Git index"
	case FilenameSourceStdin:
		modeDesc = "stdin"
	case FilenameSourceStdinNul:
		modeDesc = "stdin, NUL delimited"
	}

	if config.CheckOnly {
		modeDesc += ", check only"
	}

	if config.AckGroups {
		modeDesc += ", ack groups"
	}

	fmt.Printf("eyecue-codemap %s running (filenames from %s) ...\n", Version, modeDesc)

	var fileSources []FileSource
	var err error

	switch config.FilenameSource {
	case FilenameSourceGit:
		fileSources, err = readFilenamesFromGit()
	case FilenameSourceGitIndex:
		fileSources, err = readFilenamesFromGitIndex()
	case FilenameSourceStdin:
		fileSources, err = readFilenamesFromStdin(false)
	case FilenameSourceStdinNul:
		fileSources, err = readFilenamesFromStdin(true)
	}
	if err != nil {
		return err
	}

	fileInventory, err := inventoryFiles(config, fileSources)
	if err != nil {
		return err
	}

	// Prohibit tokens from being used in both groups and single-line locations.
	for token := range fileInventory.SinglesByToken {
		if _, ok := fileInventory.GroupsByToken[token]; ok {
			return fmt.Errorf("cannot use same token for group and single-line: %s", token)
		}
	}

	unusedTokens := make(map[string]struct{})
	var dupTokensErrs []string

	for token, tokenLocs := range fileInventory.SinglesByToken {
		if len(tokenLocs) > 1 {
			errMsg := fmt.Sprintf("duplicate token \"%s\" at:", token)
			for _, tokenLoc := range tokenLocs {
				errMsg = fmt.Sprintf("%s\n   %s:%d", errMsg, tokenLoc.filename, tokenLoc.lineNum)
			}
			dupTokensErrs = append(dupTokensErrs, errMsg)
		}

		unusedTokens[token] = struct{}{}
	}

	if len(dupTokensErrs) > 0 {
		return errors.New(strings.Join(dupTokensErrs, "\n"))
	}

	// check or update the Markdown files
	for _, fileSource := range fileInventory.MarkdownFileSources {
		err := processMarkdownFile(config, fileSource, fileInventory, unusedTokens)
		if err != nil {
			return err
		}
	}

	// show unused tokens
	var unusedTokenErrs []string
	for token := range unusedTokens {
		tokenLoc := fileInventory.SinglesByToken[token][0]
		msg := fmt.Sprintf(`unused token "%s" at %s:%d`, token, tokenLoc.filename, tokenLoc.lineNum)
		unusedTokenErrs = append(unusedTokenErrs, msg)
	}

	if len(unusedTokenErrs) > 0 {
		msg := strings.Join(unusedTokenErrs, "\n")
		if config.NoUnused {
			return errors.New(msg)
		} else {
			fmt.Println(msg)
		}
	}

	// Process groups
	if config.AckGroups {
		return ackTokenGroups(config, fileInventory)
	} else {
		return checkTokenGroups(fileInventory)
	}
}

func inventoryFiles(config Config, fileSources []FileSource) (*FileInventory, error) {
	fileInventory := &FileInventory{
		SinglesByToken: map[string][]TokenLocation{},
		GroupsByToken:  map[string][]TokenGroupInfo{},
	}

	fileSourcesCh := make(chan FileSource, len(fileSources))
	for _, fileSource := range fileSources {
		fileSourcesCh <- fileSource
	}
	close(fileSourcesCh)

	wg, wgCtx := errgroup.WithContext(context.Background())
	wg.SetLimit(runtime.GOMAXPROCS(0))
LOOP:
	for {
		select {
		case <-wgCtx.Done():
			break LOOP
		case fileSource, ok := <-fileSourcesCh:
			if !ok {
				break LOOP
			}

			wg.Go(func() error {
				err := processFile(config, fileSource, fileInventory)
				if err != nil {
					return err
				}

				if strings.ToLower(path.Ext(fileSource.Filename)) == ".md" {
					fileInventory.Lock()
					fileInventory.MarkdownFileSources = append(fileInventory.MarkdownFileSources, fileSource)
					fileInventory.Unlock()
				}

				return nil
			})
		}
	}

	err := wg.Wait()
	if err != nil {
		return nil, err
	}

	for _, groupInfos := range fileInventory.GroupsByToken {
		sort.Slice(groupInfos, func(i, j int) bool {
			if groupInfos[i].fileSource.Filename == groupInfos[j].fileSource.Filename {
				return groupInfos[i].startLineNumber < groupInfos[j].startLineNumber
			}

			return groupInfos[i].fileSource.Filename < groupInfos[j].fileSource.Filename
		})
	}

	return fileInventory, nil
}

func ackTokenGroups(config Config, fileInventory *FileInventory) error {
	groupInfosByFile := map[string][]TokenGroupInfo{}

	for _, groupInfos := range fileInventory.GroupsByToken {
		for _, groupInfo := range groupInfos {
			if groupInfo.actualHash != groupInfo.expectedHash {
				groupInfosByFile[groupInfo.fileSource.Filename] = append(groupInfosByFile[groupInfo.fileSource.Filename], groupInfo)
			}
		}
	}

	for _, groupInfos := range groupInfosByFile {
		err := ackTokenGroupsForFile(config, groupInfos)
		if err != nil {
			return err
		}
	}

	return nil
}

func ackTokenGroupsForFile(config Config, groupInfos []TokenGroupInfo) (err error) {
	fileSource := groupInfos[0].fileSource

	fileBytes, err := readFile(config, fileSource)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(fileSource.Filename, os.O_TRUNC|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
	}()

	scn := bufio.NewScanner(bytes.NewReader(fileBytes))
	scn.Split(scanLinesWithNewlines)
	currentLine := 0
	for scn.Scan() {
		currentLine++
		lineBytes := scn.Bytes()
		for _, groupInfo := range groupInfos {
			if groupInfo.endLineNumber == currentLine {
				lineBytes = tokenGroupEndRegexp.ReplaceAll(
					lineBytes,
					[]byte(fmt.Sprintf(
						"[end-%s-group:%s:%s]", tagBaseName(),
						groupInfo.token,
						groupInfo.actualHash,
					)))
			}
		}

		_, err := file.Write(lineBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

func checkTokenGroups(fileInventory *FileInventory) error {
	showAckMessage := false

	for groupName, groupInfos := range fileInventory.GroupsByToken {
		showGroup := false

		for _, groupInfo := range groupInfos {
			if groupInfo.actualHash != groupInfo.expectedHash {
				showGroup = true
				break
			}
		}

		if showGroup {
			showAckMessage = true
			sort.Slice(groupInfos, func(i, j int) bool {
				return groupInfos[i].fileSource.Filename < groupInfos[j].fileSource.Filename
			})
			fmt.Printf("group \"%s\" has changes (indicated with *):\n", groupName)
			for _, groupInfo := range groupInfos {
				indicator := " "
				if groupInfo.actualHash != groupInfo.expectedHash {
					indicator = "*"
				}

				fmt.Printf("  %s  %s:%d (lines %d-%d)\n",
					indicator,
					groupInfo.fileSource.Filename,
					groupInfo.startLineNumber,
					groupInfo.startLineNumber+1,
					groupInfo.endLineNumber-1,
				)
			}
		}
	}

	if showAckMessage {
		return errors.New(`edit groups as needed, then re-run with the "ack" argument`)
	}

	return nil
}

func shouldIncludeFile(filename string) (bool, error) {
	stat, err := os.Lstat(filename)
	if err != nil {
		return false, fmt.Errorf(`failed to stat "%s": %w`, filename, err)
	}

	if stat.Mode().IsRegular() && stat.Size() < 10*1024*1024 {
		return true, nil
	}

	return false, nil
}

func readFilenamesFromStdin(nulDelimiter bool) ([]FileSource, error) {
	if isatty.IsTerminal(os.Stdin.Fd()) {
		fmt.Println("WARNING: reading filenames from stdin. Did you forget to pipe in a list of filenames?")
	}

	var fileSources []FileSource

	scn := bufio.NewScanner(os.Stdin)
	if nulDelimiter {
		scn.Split(scanNullDelimited)
	}
	for scn.Scan() {
		filename := strings.TrimPrefix(scn.Text(), "./")

		shouldInclude, err := shouldIncludeFile(filename)
		if err != nil {
			return nil, err
		}

		if shouldInclude {
			fileSources = append(fileSources, FileSource{
				Filename:     filename,
				FromGitIndex: false,
			})
		}
	}
	if scn.Err() != nil {
		return nil, fmt.Errorf("failed to read list of filenames from stdin: %w", scn.Err())
	}

	return fileSources, nil
}

func readFilenamesFromGit() ([]FileSource, error) {
	cmd := exec.Command("git", "diff-files", "--name-only", "--diff-filter=D", "-z")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run git diff-files: %s: %w", strings.TrimSpace(string(output)), err)
	}

	deletedFilenames := make(map[string]struct{})

	for _, filenameBytes := range bytes.Split(output, []byte{0}) {
		deletedFilenames[string(filenameBytes)] = struct{}{}
	}

	cmd = exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run git ls-files: %s: %w", strings.TrimSpace(string(output)), err)
	}

	var fileSources []FileSource

	for _, filenameBytes := range bytes.Split(output, []byte{0}) {
		filename := string(filenameBytes)
		if _, isDeleted := deletedFilenames[filename]; isDeleted {
			continue
		}

		shouldInclude, err := shouldIncludeFile(filename)
		if err != nil {
			return nil, err
		}

		if shouldInclude {
			fileSources = append(fileSources, FileSource{
				Filename:     filename,
				FromGitIndex: false,
			})
		}
	}

	return fileSources, nil
}

var spacesRegexp = regexp.MustCompile(`\s+`)

func readFilenamesFromGitIndex() ([]FileSource, error) {
	// Determine which files are modified+staged and must be read from the Git index vs. the working dir
	cmd := exec.Command("git", "diff-index", "--name-only", "-z", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run git diff-index: %s: %w", strings.TrimSpace(string(output)), err)
	}

	stagedFilenames := make(map[string]struct{})

	for _, filenameBytes := range bytes.Split(output, []byte{0}) {
		stagedFilenames[string(filenameBytes)] = struct{}{}
	}

	// Get a list of all filenames in the Git index
	cmd = exec.Command("git", "ls-files", "--stage", "-z")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run git ls-files: %s: %w", strings.TrimSpace(string(output)), err)
	}

	var fileSources []FileSource

	for _, lineBytes := range bytes.Split(output, []byte{0}) {
		line := string(lineBytes)

		// filter out non-files
		if !strings.HasPrefix(line, "100") {
			continue
		}

		// Output looks like:
		// 100644 b438169c25a6cf5649e09d8d51092998fa4e904e 0	Dockerfile
		parts := spacesRegexp.Split(line, 4)
		filename := parts[3]
		_, isStaged := stagedFilenames[filename]
		fileSources = append(fileSources, FileSource{
			Filename:     filename,
			FromGitIndex: isStaged,
		})
	}

	return fileSources, nil
}

func readFileFromGitIndex(filename string) ([]byte, error) {
	cmd := exec.Command("git", "show", ":"+filename)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git show :%s failed: %w", filename, err)
	}

	return output, err
}

func readFile(config Config, fileSource FileSource) ([]byte, error) {
	if fileSource.FromGitIndex {
		if config.Verbose {
			fmt.Printf("git index: reading \"%s\"\n", fileSource.Filename)
		}
		return readFileFromGitIndex(fileSource.Filename)
	}

	if config.Verbose {
		fmt.Printf("working dir: reading \"%s\"\n", fileSource.Filename)
	}
	return os.ReadFile(fileSource.Filename)
}

func processTokenGroups(fileSource FileSource, fileBytes []byte, fileInventory *FileInventory) error {
	type CurrentGroup struct {
		Hasher          hash.Hash
		Token           string
		StartLineNumber int
	}
	var currentGroup *CurrentGroup

	currentLine := 1

	scn := bufio.NewScanner(bytes.NewReader(fileBytes))
	scn.Split(scanLinesWithNewlines)
	for scn.Scan() {
		line := scn.Text()

		groupMatch := tokenGroupEndRegexp.FindStringSubmatch(line)
		if len(groupMatch) > 0 {
			token := groupMatch[1]
			expectedHash := groupMatch[3]

			if currentGroup == nil {
				return fmt.Errorf(`end-%s-group for unknown group "%s" (%s:%d)`, tagBaseName(), token, fileSource.Filename, currentLine)
			}

			fileInventory.Lock()
			fileInventory.GroupsByToken[token] = append(fileInventory.GroupsByToken[token], TokenGroupInfo{
				token:           token,
				fileSource:      fileSource,
				startLineNumber: currentGroup.StartLineNumber,
				endLineNumber:   currentLine,
				actualHash:      fmt.Sprintf("%x", currentGroup.Hasher.Sum(nil)),
				expectedHash:    expectedHash,
			})
			fileInventory.Unlock()

			currentGroup = nil
		}

		if currentGroup != nil {
			_, err := currentGroup.Hasher.Write(scn.Bytes())
			if err != nil {
				return err
			}
		}

		groupMatch = tokenGroupStartRegexp.FindStringSubmatch(line)
		if len(groupMatch) > 0 {
			token := groupMatch[1]
			if currentGroup != nil {
				return fmt.Errorf(`overlapping %s-group "%s" not allowed (%s:%d)`, tagBaseName(), token, fileSource.Filename, currentLine)
			}

			currentGroup = &CurrentGroup{
				Hasher:          sha1.New(),
				Token:           token,
				StartLineNumber: currentLine,
			}
		}

		currentLine++
	}

	if currentGroup != nil {
		return fmt.Errorf("unclosed %s-group in %s", tagBaseName(), fileSource.Filename)
	}

	return nil
}

func processFile(config Config, fileSource FileSource, fileInventory *FileInventory) error {
	for _, ext := range ignoreExtensions {
		if strings.HasSuffix(fileSource.Filename, ext) {
			return nil
		}
	}

	fileBytes, err := readFile(config, fileSource)
	if err != nil {
		return fmt.Errorf(`failed to read "%s": %w`, fileSource.Filename, err)
	}

	// generate tokens
	if !config.CheckOnly {
		changed := false
		fileBytes = tokenNeededRegexp.ReplaceAllFunc(fileBytes, func(matched []byte) []byte {
			changed = true
			token := generateToken()
			fmt.Printf("Added new token \"%s\" to \"%s\"\n", token, fileSource.Filename)
			return []byte("[" + string(matched[1:len(matched)-1]) + ":" + token + "]")
		})

		if changed {
			err := os.WriteFile(fileSource.Filename, fileBytes, 0)
			if err != nil {
				return fmt.Errorf(`failed to write "%s": %w`, fileSource.Filename, err)
			}
		}
	}

	err = processTokenGroups(fileSource, fileBytes, fileInventory)
	if err != nil {
		return err
	}

	// We'll link to the entire file (instead of a specific line) for any [eyecue-codemap] that:
	// * Is preceded only by the shebang and/or blank lines
	// * Is followed by a blank line or EOF
	linkToFile := true

	// inventory tokens
	currentLine := 1
	var line string
	var peekLine bool
	scn := bufio.NewScanner(bytes.NewReader(fileBytes))
	for {
		if peekLine {
			peekLine = false
		} else {
			if !scn.Scan() {
				break
			}
			line = scn.Text()
		}

		doneScanning := false

		m := tokenRegexp.FindAllStringSubmatch(line, -1)

		// If we found a token, and we still think we want to link to the file,
		// check the next line to make sure it's blank or EOF.
		if linkToFile && len(m) > 0 {
			if scn.Scan() {
				peekLine = true
				line = scn.Text()
				if strings.TrimSpace(line) != "" {
					linkToFile = false
				}
			} else {
				// no more lines, we'll link to the file
				doneScanning = true
			}
		}

		for _, match := range m {
			before := strings.TrimSpace(match[1])
			token := match[2]
			after := strings.TrimSpace(match[3])

			// If the only thing on the line is the codemap comment,
			// link to the next line. Add more comment strings here as needed.
			lineNum := currentLine
			if after == "" && (before == "//" || before == "#") {
				lineNum++
			} else if before == "<!--" && after == "-->" {
				lineNum++
			}

			fileInventory.Lock()
			fileInventory.SinglesByToken[token] = append(fileInventory.SinglesByToken[token], TokenLocation{
				filename:   fileSource.Filename,
				lineNum:    lineNum,
				linkToFile: linkToFile,
			})
			fileInventory.Unlock()
		}

		if doneScanning {
			break
		}

		if linkToFile && !(strings.HasPrefix(line, "#!") || strings.TrimSpace(line) == "") {
			linkToFile = false
		}

		currentLine++
	}
	if scn.Err() != nil {
		if scn.Err() == bufio.ErrTooLong {
			// probably not a text file. This is OK.
			return nil
		}

		return fmt.Errorf(`failed to scan "%s": %w`, fileSource.Filename, scn.Err())
	}

	return nil
}

type markdownContext struct {
	Changed       bool
	CheckErrors   []string
	CheckOny      bool
	FileBytes     []byte
	FileInventory *FileInventory
	Filename      string
	FilenameDir   string
	UnusedTokens  map[string]struct{}
}

func processMarkdownFile(config Config, mdFileSource FileSource, fileInventory *FileInventory, unusedTokens map[string]struct{}) error {
	fileBytes, err := readFile(config, mdFileSource)
	if err != nil {
		return fmt.Errorf(`failed to read "%s": %w`, mdFileSource.Filename, err)
	}

	mdContext := &markdownContext{
		CheckOny:      config.CheckOnly,
		FileBytes:     fileBytes,
		FileInventory: fileInventory,
		Filename:      mdFileSource.Filename,
		FilenameDir:   filepath.Dir(mdFileSource.Filename),
		UnusedTokens:  unusedTokens,
	}

	err = updateTokenRefs(mdContext)
	if err != nil {
		return err
	}

	err = renderMarkdownGroupTemplates(mdContext)
	if err != nil {
		return err
	}

	if config.CheckOnly {
		if len(mdContext.CheckErrors) > 0 {
			fmt.Println(strings.Join(mdContext.CheckErrors, "\n"))
		}
	} else if mdContext.Changed {
		err := os.WriteFile(mdFileSource.Filename, mdContext.FileBytes, 0)
		if err != nil {
			return fmt.Errorf(`failed to write "%s": %w`, mdFileSource.Filename, err)
		}
	}

	return nil
}

func updateTokenRefs(mdContext *markdownContext) error {
	var resultBuf bytes.Buffer

	fileBytes := mdContext.FileBytes

	lastLine := false
	for lineNum := 1; !lastLine; lineNum++ {
		var lineBytes []byte
		newLineIndex := bytes.IndexByte(fileBytes, '\n')
		if newLineIndex == -1 {
			lineBytes = fileBytes
			lastLine = true
		} else {
			lineBytes = fileBytes[:newLineIndex+1]
			fileBytes = fileBytes[newLineIndex+1:]
		}

		lineBytes = tokenRefRegexp.ReplaceAllFunc(lineBytes, func(m []byte) []byte {
			tokenIndex := bytes.IndexByte(m, ':') + 1
			tokenEndIndex := tokenIndex + bytes.IndexByte(m[tokenIndex:], '-')
			token := string(m[tokenIndex:tokenEndIndex])

			tokenLocs := mdContext.FileInventory.SinglesByToken[token]
			if len(tokenLocs) == 0 {
				mdContext.CheckErrors = append(mdContext.CheckErrors, fmt.Sprintf(`token "%s" at "%s:%d" was not found`, token, mdContext.Filename, lineNum))
			} else {
				delete(mdContext.UnusedTokens, token)

				loc := tokenLocs[0]
				locRelPath, err := filepath.Rel(mdContext.FilenameDir, loc.filename)
				if err != nil {
					panic(err)
				} else {
					original := string(m)
					var mdTarget string
					var outputTarget string
					if loc.linkToFile {
						mdTarget = locRelPath
						outputTarget = locRelPath
					} else {
						mdTarget = fmt.Sprintf("%s#L%d", locRelPath, loc.lineNum)
						outputTarget = fmt.Sprintf("%s:%d", locRelPath, loc.lineNum)
					}
					replacement := fmt.Sprintf("<!--%s:%s-->](%s)", tagBaseName(), token, mdTarget)
					if original != replacement {
						if mdContext.CheckOny {
							mdContext.CheckErrors = append(mdContext.CheckErrors, fmt.Sprintf(`incorrect link at "%s:%d" token "%s"`, mdContext.Filename, lineNum, token))
						} else {
							mdContext.Changed = true
							fmt.Printf("updated link at \"%s:%d\" token \"%s\" -> \"%s\"\n", mdContext.Filename, lineNum, token, outputTarget)
							return []byte(replacement)
						}
					}
				}
			}

			return m
		})

		_, err := resultBuf.Write(lineBytes)
		if err != nil {
			return err
		}
	}

	mdContext.FileBytes = resultBuf.Bytes()

	return nil
}

func renderMarkdownGroupTemplates(mdContext *markdownContext) error {
	var resultBuf bytes.Buffer

	remainingIndex := 0

	matches := tokenGroupRefRegexp.FindAllSubmatchIndex(mdContext.FileBytes, -1)
	for _, match := range matches {
		_, err := resultBuf.Write(mdContext.FileBytes[remainingIndex:match[0]])
		if err != nil {
			return err
		}

		remainingIndex = match[1]
		startTag := mdContext.FileBytes[match[2]:match[3]]
		token := string(mdContext.FileBytes[match[4]:match[5]])
		templateText := string(mdContext.FileBytes[match[6]:match[7]])
		existingContent := mdContext.FileBytes[match[8]:match[9]]
		endTag := mdContext.FileBytes[match[10]:match[11]]

		groupInfos := mdContext.FileInventory.GroupsByToken[token]

		if len(groupInfos) == 0 {
			// TODO: get the line number into this error message
			mdContext.CheckErrors = append(mdContext.CheckErrors, fmt.Sprintf(`group token "%s" in "%s" was not found`, token, mdContext.Filename))
			continue
		}

		tpl, err := template.New("").Parse(templateText)
		if err != nil {
			return err
		}

		type GroupTemplateData struct {
			File              string
			Line              int
			FileLine          string
			RangeHref         string
			MarkdownRangeLink string
		}

		templateData := make([]GroupTemplateData, len(groupInfos))
		for i, groupInfo := range groupInfos {
			fileLine := fmt.Sprintf("%s:%d", groupInfo.fileSource.Filename, groupInfo.startLineNumber)

			locRelPath, err := filepath.Rel(mdContext.FilenameDir, groupInfo.fileSource.Filename)
			if err != nil {
				return err
			}

			rangeHref := fmt.Sprintf(
				"%s#L%d-L%d",
				locRelPath,
				groupInfo.startLineNumber+1,
				groupInfo.endLineNumber-1,
			)

			markdownRangeLink := fmt.Sprintf("[%s](%s)", fileLine, rangeHref)

			templateData[i] = GroupTemplateData{
				File:              groupInfo.fileSource.Filename,
				Line:              groupInfo.startLineNumber,
				FileLine:          fileLine,
				RangeHref:         rangeHref,
				MarkdownRangeLink: markdownRangeLink,
			}
		}

		_, err = resultBuf.Write(startTag)
		if err != nil {
			return err
		}
		err = resultBuf.WriteByte('\n')
		if err != nil {
			return err
		}

		var templateOutputBuf bytes.Buffer
		err = tpl.Execute(&templateOutputBuf, templateData)
		if err != nil {
			return err
		}

		if !bytes.Equal(templateOutputBuf.Bytes(), existingContent) {
			mdContext.Changed = true

			if mdContext.CheckOny {
				mdContext.CheckErrors = append(mdContext.CheckErrors, fmt.Sprintf(`incorrect group "%s" template content in "%s"`, token, mdContext.Filename))
			} else {
				fmt.Printf(`updating group "%s" template content in "%s"`+"\n", token, mdContext.Filename)
			}
		}

		_, err = resultBuf.Write(templateOutputBuf.Bytes())
		if err != nil {
			return err
		}

		_, err = resultBuf.Write(endTag)
		if err != nil {
			return err
		}
	}

	_, err := resultBuf.Write(mdContext.FileBytes[remainingIndex:])
	if err != nil {
		return err
	}

	mdContext.FileBytes = resultBuf.Bytes()
	return nil
}

func generateToken() string {
	buf := make([]byte, 8)
	_, err := rand.Read(buf)
	if err != nil {
		panic(fmt.Errorf("failed to read random bytes: %w", err))
	}

	return base58.Encode(buf)
}
