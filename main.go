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

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println("Usage: eyecue-codemap [--check-only]\nPipe in a list of filenames to stdin, one per line.")
			os.Exit(0)
		}
	}

	if len(os.Args) > 2 {
		fmt.Println("ERROR: too many arguments")
		os.Exit(2)
	}

	checkOnly := false
	if len(os.Args) == 2 {
		if os.Args[1] == "--check-only" {
			checkOnly = true
		} else {
			fmt.Printf("ERROR: unrecognized argument: %s\n", os.Args[1])
			os.Exit(2)
		}
	}

	if isatty.IsTerminal(os.Stdin.Fd()) {
		fmt.Println("WARNING: reading filenames from stdin. Did you forget to pipe in a list of filenames?")
	}

	fmt.Printf("eyecue-codemap running...")
	if checkOnly {
		fmt.Printf(" (check only)")
	}
	fmt.Println()

	err := run(os.Stdin, checkOnly)
	if err != nil {
		fmt.Printf("eyecue-codemap error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("eyecue-codemap completed successfully")
}

func run(filenameSource io.Reader, checkOnly bool) error {
	tokenMap := make(TokenMap)
	var mdFilenames []string

	// inventory each input file, generating tokens and updating them if needed
	scn := bufio.NewScanner(filenameSource)
	for scn.Scan() {
		filename := scn.Text()
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

	// detect duplicate tokens
	for token, tokenLocs := range tokenMap {
		if len(tokenLocs) > 1 {
			errMsg := fmt.Sprintf("duplicate token \"%s\" at:", token)
			for _, tokenLoc := range tokenLocs {
				errMsg = fmt.Sprintf("%s\n   %s:%d", errMsg, tokenLoc.filename, tokenLoc.lineNum)
			}
			return errors.New(errMsg)
		}
	}

	// update the Markdown files
	for _, filename := range mdFilenames {
		err := updateMarkdownFile(filename, tokenMap, checkOnly)
		if err != nil {
			return err
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

func updateMarkdownFile(filename string, tokenMap TokenMap, checkOnly bool) error {
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf(`failed to read "%s": %w`, filename, err)
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
			replaceErr = fmt.Errorf(`token "%s" in "%s" was not found`, token, filename)
		} else {
			loc := tokenLocs[0]
			original := string(m)
			replacement := fmt.Sprintf("<!--eyecue-codemap:%s-->](%s#L%d)", token, loc.filename, loc.lineNum)
			if original != replacement {
				if checkOnly {
					replaceErr = fmt.Errorf(`incorrect link in "%s" to token "%s"`, filename, token)
				} else {
					changed = true
					fmt.Printf("Updated link in \"%s\" to token \"%s\" -> \"%s:%d\"\n", filename, token, loc.filename, loc.lineNum)
					return []byte(replacement)
				}
			}
		}

		return m
	})

	if replaceErr != nil {
		return replaceErr
	}

	if changed {
		err := os.WriteFile(filename, fileBytes, 0)
		if err != nil {
			return fmt.Errorf(`failed to write "%s": %w`, filename, err)
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
