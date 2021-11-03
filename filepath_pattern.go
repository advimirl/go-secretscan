package main

const PartPath string = "path"

type FilepathFullPattern struct {
	FullPattern
}

func (s FilepathFullPattern) match(file MatchFile) (bool, string, string) {
	m, c := s.strMatch(file.Path)
	return m, PartPath, c
}

type FilepathMatchPattern struct {
	MatchPattern
}

func (s FilepathMatchPattern) match(file MatchFile) (bool, string, string) {
	m, c := s.rexMatch(file.Extension)
	return m, PartPath, c
}
