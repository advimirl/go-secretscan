package main

import (
	"github.com/sirupsen/logrus"
	"regexp"
	"regexp/syntax"
)

type Pattern interface {
	Name() string
	match(file MatchFile) (bool, string, string)
	getContentsMatches(contents []byte) []string
}

func createPattern(signature SignatureServiceRecord, blacklistedStringsRef *[]string) Pattern {
	if signature.Match != "" {
		pattern := FullPattern{
			match:      signature.Match,
			name:       signature.Name,
			confidence: signature.Confidence,
		}
		switch signature.Part {
		case PartExtension:
			return ExtensionFullPattern{pattern}
		case PartFilename:
			return FilenameFullPattern{pattern}
		case PartPath:
			return FilepathFullPattern{pattern}
		default:
			logrus.Panicf("%v", signature)
		}
	} else {
		if _, err := syntax.Parse(signature.Match, syntax.FoldCase); err == nil {
			pattern := MatchPattern{
				name:               signature.Name,
				match:              regexp.MustCompile(signature.Regex),
				confidence:         signature.Confidence,
				blacklistedStrings: blacklistedStringsRef,
			}
			switch signature.Part {
			case PartExtension:
				return ExtensionMatchPattern{pattern}
			case PartFilename:
				return FilenameMatchPattern{pattern}
			case PartPath:
				return FilepathMatchPattern{pattern}
			case PartContents:
				return ContentMatchPattern{pattern}
			default:
				panic(signature)
			}
		}

	}
	return nil
}
