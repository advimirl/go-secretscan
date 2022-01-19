package main

import (
	"fmt"
	"strings"
	"time"

	"gitlab.com/gitlab-org/security-products/analyzers/report/v2"
)

const (
	scannerID      = "go-secretscan"
	scannerName    = "go-secretscan"
	scannerURL     = "-"
	scannerVendor  = "1n0guard"
	scannerVersion = "0.1-p6"
	analyzerId     = "go-sscan"
)

var secretsScaner = report.Scanner{
	ID:   scannerID,
	Name: scannerName,
}

var secretsScannerDetailed = report.ScannerDetails{
	ID:      scannerID,
	Name:    scannerName,
	URL:     scannerURL,
	Vendor:  report.Vendor{Name: scannerVendor},
	Version: scannerVersion,
}

func createReport(messages []*Message, storage *Storage) (*report.Report, error) {
	var vulns []report.Vulnerability

	for _, msg := range messages {
		var commit report.Commit
		if msg.commitInfo != nil {
			commit = report.Commit{
				Author:  msg.commitInfo.Author,
				Date:    msg.commitInfo.Date,
				Message: msg.commitInfo.Message,
				Sha:     msg.commitInfo.Sha,
			}
		}
		confidence := report.ConfidenceLevelUnknown
		if msg.Confidence != "" {
			switch strings.ToLower(msg.Confidence) {
			case "high":
				confidence = report.ConfidenceLevelHigh
			case "medium":
				confidence = report.ConfidenceLevelMedium
			}
		}
		vulns = append(vulns, report.Vulnerability{
			Category: report.CategorySecretDetection,
			Scanner:  secretsScaner,
			Name:     msg.MatchName,
			Message:  msg.MatchName,
			Description: fmt.Sprintf("Match based by rule of `%s` type.\nFull path to file: %s\nConfidence: %s\n",
				msg.MatchType, msg.MatchURL, msg.Confidence),
			RawSourceCodeExtract: msg.RawMatchContent,
			Severity:             report.SeverityLevelCritical,
			Confidence:           confidence,
			Location: report.Location{File: msg.Filename,
				LineStart: msg.MatchedLineNumbers,
				Commit:    &commit},
			Identifiers: nil,
			Links:       nil,
		})
	}
	endTime := report.ScanTime(time.Now())
	rp := report.NewReport()
	rp.Analyzer = analyzerId
	//rp.Config.Path = storage.getConfigPath()
	rp.Vulnerabilities = vulns
	rp.Scan.EndTime = &endTime
	rp.Scan.Scanner = secretsScannerDetailed

	return &rp, nil
}
