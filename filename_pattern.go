package main

const PartFilename string = "filename"

type FilenameFullPattern struct {
	FullPattern
}

func (s FilenameFullPattern) match(file MatchFile) (bool, string, string) {
	m, c := s.strMatch(file.Filename)
	return m, PartFilename, c
}

type FilenameMatchPattern struct {
	MatchPattern
}

func (s FilenameMatchPattern) match(file MatchFile) (bool, string, string) {
	m, c := s.rexMatch(file.Filename)
	return m, PartFilename, c
}
