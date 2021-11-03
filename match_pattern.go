package main

import (
	"regexp"
	"strings"
)

type MatchPattern struct {
	Pattern
	match              *regexp.Regexp
	name               string
	confidence         string
	blacklistedStrings *[]string
}

func (s MatchPattern) getContentsMatches(contents []byte) []string {
	matches := make([]string, 0)

	for _, match := range s.match.FindAllSubmatch(contents, -1) {
		match := string(match[0])
		blacklistedMatch := false

		for _, blacklistedString := range *s.blacklistedStrings {
			if strings.Contains(strings.ToLower(match), strings.ToLower(blacklistedString)) {
				blacklistedMatch = true
			}
		}

		if !blacklistedMatch {
			matches = append(matches, match)
		}
	}

	return matches
}

func (s MatchPattern) rexMatch(stringToMatch string) (bool, string) {
	return s.match.MatchString(stringToMatch), s.confidence
}

func (s MatchPattern) rexMatchBytes(bytesToMatch []byte) (bool, string) {
	return s.match.Match(bytesToMatch), s.confidence
}

func (s MatchPattern) Name() string {
	return s.name
}
