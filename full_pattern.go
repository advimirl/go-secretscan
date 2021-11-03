package main

type FullPattern struct {
	Pattern
	match      string
	name       string
	confidence string
}

func (s FullPattern) getContentsMatches(_ []byte) []string {
	return nil
}

func (s FullPattern) Name() string {
	return s.name
}

func (s FullPattern) strMatch(stringToMatch string) (bool, string) {
	return s.match == stringToMatch, s.confidence
}
