package main

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

type MatchProcessorFunc func(file MatchFile) bool

type MatchFile struct {
	Path       string
	Filename   string
	Extension  string
	Contents   []byte
	CommitInfo *CommitInfo
	Checker    *Checker
}

type MatchRecord struct {
	MatchType          string
	MatchedContent     string
	MatchedTimes       int
	MatchedFilename    string
	MatchedLineNumbers int
	Confidence         string
}

func newMatchFile(path string, content []byte, info *CommitInfo, checker *Checker) MatchFile {
	path = filepath.ToSlash(path)
	_, filename := filepath.Split(path)
	extension := filepath.Ext(path)

	return MatchFile{
		Path:       path,
		Filename:   filename,
		Extension:  extension,
		Contents:   content,
		CommitInfo: info,
		Checker:    checker,
	}
}

func getMatch(file MatchFile, funcs ...MatchProcessorFunc) (MatchRecord, string, bool) {
	var (
		isMatched            = false
		record               MatchRecord
		matchedSignatureName string
		storage              = file.Checker.storage
	)
	fnMatched := true
	for _, fn := range funcs {
		fnMatched = fnMatched && fn(file)
	}
	if !fnMatched {
		return record, "", false
	}
	for _, pattern := range storage.getPatterns() {
		if matched, part, confidence := pattern.match(file); matched {
			isMatched = true
			var matches []string
			matches = pattern.getContentsMatches(file.Contents)
			lineNum := 0
			findings := ""
			if matches != nil && len(matches) > 0 {
				lineNum = findLineOnFile(file.Contents, matches[0])
				findings = strings.Join(matches, ",\n")
			}
			record = MatchRecord{part, findings, len(matches), file.Path, lineNum, confidence}
			return record, pattern.Name(), true
		}
	}
	if file.Checker.checkEntropyFile(file.Filename, file.Extension) {
		scanner := bufio.NewScanner(bytes.NewReader(file.Contents))
		lineNum := 1
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) > 6 && len(line) < 30 {
				for _, entropySearch := range entropySearches {
					entropy := getEntropy(line, entropySearch)
					if entropy >= entropySearch.EntropyBorder {
						blacklistedMatch := false
						for _, blacklistedString := range storage.getBlacklistedStrings() {
							if strings.Contains(strings.ToLower(line), strings.ToLower(blacklistedString)) {
								blacklistedMatch = true
							}
						}
						if !blacklistedMatch {
							logrus.Printf("High entropy line found in %s. Line: %s. Path: %s", file.Filename, line, file.Path)
							record = MatchRecord{entropySearch.RuleName, line, 1, file.Path, lineNum, "high"}
							isMatched = true
							break
						}
					}
				}
			}
			lineNum++
		}
	}
	return record, matchedSignatureName, isMatched
}
