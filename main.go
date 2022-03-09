package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/akamensky/base58"
	"github.com/mattn/go-isatty"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type TokenLocation struct {
	filename string
	lineNum  int
}

type TokenMap = map[string][]TokenLocation

var tokenNeededRegexp = regexp.MustCompile(`\[eyecue-codemap]`)
var tokenRegexp = regexp.MustCompile(`^(.*)\[eyecue-codemap:([A-Za-z0-9]+)](.*)$`)
var tokenRefRegexp = regexp.MustCompile(`<!--eyecue-codemap:[A-Za-z0-9]+-->]\(.*?\)`)

var ignoreExtensions = []string{
	".jpg",
	".jpeg",
	".png",
}

var Version string = "dev"

func main() {
	checkOnly := false
	noUnused := false

	for _, arg := range os.Args[1:] {
		switch arg {
		case "--help", "-h":
			fmt.Printf("eyecue-codemap version %s\n"+
				"Usage: eyecue-codemap [--check-only] [--no-unused]\n"+
				"Pipe in a list of filenames to stdin, one per line.\n", Version)
			os.Exit(0)
		case "--check-only":
			checkOnly = true
		case "--no-unused":
			noUnused = true
		default:
			fmt.Printf("ERROR: unrecognized argument: %s\n", arg)
			os.Exit(2)
		}
	}

	if isatty.IsTerminal(os.Stdin.Fd()) {
		fmt.Println("WARNING: reading filenames from stdin. Did you forget to pipe in a list of filenames?")
	}

	fmt.Printf("eyecue-codemap %s running...", Version)
	if checkOnly {
		fmt.Printf(" (check only)")
	}
	fmt.Println()

	err := run(os.Stdin, checkOnly, noUnused)
	if err != nil {
		fmt.Printf("eyecue-codemap error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("eyecue-codemap completed successfully")
}

func run(filenameSource io.Reader, checkOnly bool, noUnused bool) error {
	tokenMap := make(TokenMap)
	var mdFilenames []string

	// inventory each input file, generating tokens and updating them if needed
	scn := bufio.NewScanner(filenameSource)
SCAN:
	for scn.Scan() {
		filename := scn.Text()

		for _, ext := range ignoreExtensions {
			if strings.HasSuffix(filename, ext) {
				continue SCAN
			}
		}

		err := processFile(filename, tokenMap, checkOnly)
		if err != nil {
			return err
		}

		if strings.ToLower(path.Ext(filename)) == ".md" {
			mdFilenames = append(mdFilenames, filename)
		}
	}
	if scn.Err() != nil {
		return fmt.Errorf("failed to read list of filenames: %w", scn.Err())
	}

	unusedTokens := make(map[string]struct{})

	for token, tokenLocs := range tokenMap {
		// detect duplicate tokens
		if len(tokenLocs) > 1 {
			errMsg := fmt.Sprintf("duplicate token \"%s\" at:", token)
			for _, tokenLoc := range tokenLocs {
				errMsg = fmt.Sprintf("%s\n   %s:%d", errMsg, tokenLoc.filename, tokenLoc.lineNum)
			}
			return errors.New(errMsg)
		}

		unusedTokens[token] = struct{}{}
	}

	// update the Markdown files
	for _, filename := range mdFilenames {
		err := updateMarkdownFile(filename, tokenMap, checkOnly, unusedTokens)
		if err != nil {
			return err
		}
	}

	// show unused tokens
	for token := range unusedTokens {
		tokenLoc := tokenMap[token][0]
		msg := fmt.Sprintf("unused token %s at %s:%d", token, tokenLoc.filename, tokenLoc.lineNum)
		if noUnused {
			return errors.New(msg)
		} else {
			fmt.Println(msg)
		}
	}

	return nil
}

func processFile(filename string, tokenMap TokenMap, checkOnly bool) error {
	stat, err := os.Lstat(filename)
	if err != nil {
		return fmt.Errorf(`failed to stat "%s": %w`, filename, err)
	}
	if !stat.Mode().IsRegular() || stat.Size() > 10*1024*1024 {
		return nil
	}

	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf(`failed to read "%s": %w`, filename, err)
	}

	// generate tokens
	if !checkOnly {
		changed := false
		fileBytes = tokenNeededRegexp.ReplaceAllFunc(fileBytes, func(_ []byte) []byte {
			changed = true
			token := generateToken()
			fmt.Printf("Added new token \"%s\" to \"%s\"\n", token, filename)
			return []byte("[eyecue-codemap:" + token + "]")
		})

		if changed {
			err := os.WriteFile(filename, fileBytes, 0)
			if err != nil {
				return fmt.Errorf(`failed to write "%s": %w`, filename, err)
			}
		}
	}

	// inventory tokens
	currentLine := 1
	scn := bufio.NewScanner(bytes.NewReader(fileBytes))
	for scn.Scan() {
		m := tokenRegexp.FindAllSubmatch(scn.Bytes(), -1)
		for _, match := range m {
			before := strings.TrimSpace(string(match[1]))
			token := string(match[2])
			after := strings.TrimSpace(string(match[3]))

			// If the only thing on the line is the codemap comment,
			// link to the next line. Add more comment strings here as needed.
			lineNum := currentLine
			if after == "" && (before == "//" || before == "#") {
				lineNum++
			} else if before == "<!--" && after == "-->" {
				lineNum++
			}

			tokenMap[token] = append(tokenMap[token], TokenLocation{
				filename: filename,
				lineNum:  lineNum,
			})
		}
		currentLine++
	}
	if scn.Err() != nil {
		if scn.Err() == bufio.ErrTooLong {
			// probably not a text file. This is OK.
			return nil
		}

		return fmt.Errorf(`failed to scan "%s": %w`, filename, scn.Err())
	}

	return nil
}

func updateMarkdownFile(mdFilename string, tokenMap TokenMap, checkOnly bool, unusedTokens map[string]struct{}) error {
	mdFilenameDir := filepath.Dir(mdFilename)
	fileBytes, err := os.ReadFile(mdFilename)
	if err != nil {
		return fmt.Errorf(`failed to read "%s": %w`, mdFilename, err)
	}

	var replaceErr error
	changed := false
	fileBytes = tokenRefRegexp.ReplaceAllFunc(fileBytes, func(m []byte) []byte {
		if replaceErr != nil {
			return fileBytes
		}

		tokenIndex := bytes.IndexByte(m, ':') + 1
		tokenEndIndex := tokenIndex + bytes.IndexByte(m[tokenIndex:], '-')
		token := string(m[tokenIndex:tokenEndIndex])

		tokenLocs := tokenMap[token]
		if len(tokenLocs) == 0 {
			replaceErr = fmt.Errorf(`token "%s" in "%s" was not found`, token, mdFilename)
		} else {
			delete(unusedTokens, token)

			loc := tokenLocs[0]
			locRelPath, err := filepath.Rel(mdFilenameDir, loc.filename)
			if err != nil {
				replaceErr = fmt.Errorf("filepath.Rel(%s, %s): %w", mdFilenameDir, loc.filename, err)
			} else {
				original := string(m)
				replacement := fmt.Sprintf("<!--eyecue-codemap:%s-->](%s#L%d)", token, locRelPath, loc.lineNum)
				if original != replacement {
					if checkOnly {
						replaceErr = fmt.Errorf(`incorrect link in "%s" to token "%s"`, mdFilename, token)
					} else {
						changed = true
						fmt.Printf("Updated link in \"%s\" to token \"%s\" -> \"%s:%d\"\n", mdFilename, token, locRelPath, loc.lineNum)
						return []byte(replacement)
					}
				}
			}
		}

		return m
	})

	if replaceErr != nil {
		return replaceErr
	}

	if changed {
		err := os.WriteFile(mdFilename, fileBytes, 0)
		if err != nil {
			return fmt.Errorf(`failed to write "%s": %w`, mdFilename, err)
		}
	}

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
