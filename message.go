package main

type Message struct {
	ProjectID          int    `json:"project_id"`
	ProjectName        string `json:"project_name"`
	MatchURL           string `json:"match_url"`
	Path               string `json:"path"`
	Filename           string `json:"filename"`
	MatchType          string `json:"match_type"`
	MatchName          string `json:"match_name"`
	RawMatchContent    string `json:"-"`
	Confidence         string `json:"confidence"`
	MatchedLineNumbers int
	commitInfo         *CommitInfo
}
