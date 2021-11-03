package main

const PartContents string = "contents"

type ContentMatchPattern struct {
	MatchPattern
}

func (s ContentMatchPattern) match(file MatchFile) (bool, string, string) {
	m, c := s.rexMatchBytes(file.Contents)
	return m, PartContents, c

}
