package main

const PartExtension string = "extension"

type ExtensionFullPattern struct {
	FullPattern
}

func (s ExtensionFullPattern) match(file MatchFile) (bool, string, string) {
	m, c := s.strMatch(file.Extension)
	return m, PartExtension, c
}

type ExtensionMatchPattern struct {
	MatchPattern
}

func (s ExtensionMatchPattern) match(file MatchFile) (bool, string, string) {
	m, c := s.rexMatch(file.Extension)
	return m, PartExtension, c
}
