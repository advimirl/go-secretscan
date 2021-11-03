package main

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"os"
	"path"
	"sync"
	"time"
)

type SignatureServiceStorage struct {
	Signatures []SignatureServiceRecord `yaml:"signatures"`
}

type SignatureServiceRecord struct {
	Name       string `yaml:"name"`
	Part       string `yaml:"part"`
	Match      string `yaml:"match,omitempty"`
	Regex      string `yaml:"regex,omitempty"`
	Confidence string `yaml:"confidence,omitempty"`
	Verifier   string `yaml:"verifier,omitempty"`
}

type SignatureService struct {
	signatures             []Pattern
	signatureFilePath      string
	signatureFileTimestamp time.Time
	updateLock             sync.Mutex
}

func createSignatureService(signatureFilename string, blacklistedStringsRef *[]string) *SignatureService {
	signatureService := &SignatureService{}
	signatureService.readFromFile(signatureFilename, blacklistedStringsRef)
	stat, err := os.Stat(signatureService.signatureFilePath)
	if err != nil {
		panic(err)
	}
	signatureService.signatureFileTimestamp = stat.ModTime()
	signatureService.updateLock = sync.Mutex{}

	return signatureService
}

func (signatureService *SignatureService) readFromFile(signatureFilename string, blacklistedStringsRef *[]string) {
	signServiceStorage := &SignatureServiceStorage{}
	data, fullPath, err := readAndLocateFile("", signatureFilename)
	if err != nil {
		panic(err)
	}
	signatureService.signatureFilePath = fullPath
	err = yaml.Unmarshal(data, signServiceStorage)
	if err != nil {
		panic(err)
	}
	var patterns []Pattern
	for _, signature := range signServiceStorage.Signatures {
		if pattern := createPattern(signature, blacklistedStringsRef); pattern != nil {
			patterns = append(patterns, pattern)
		}
	}
	signatureService.signatures = patterns
}

func (signatureService *SignatureService) getPatterns(blacklistedStringsRef *[]string) []Pattern {
	info, err := os.Stat(signatureService.signatureFilePath)
	if err != nil {
		panic(err)
	}
	if info.ModTime().After(signatureService.signatureFileTimestamp) {
		signatureService.updateLock.Lock()
		logrus.Println("Updating signatures....")
		signatureService.readFromFile(path.Base(signatureService.signatureFilePath), blacklistedStringsRef)
		signatureService.signatureFileTimestamp = info.ModTime()
		signatureService.updateLock.Unlock()
	}
	return signatureService.signatures
}
