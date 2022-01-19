package main

import (
	"github.com/sirupsen/logrus"
	"github.com/doublestraus/go-dodjo"
	"os"
	"runtime"
)

type Storage struct {
	config           *Config
	dodjo            *DodjoOpts
	signatureService *SignatureService
	reportsDir       string
}

func createStorage(options *Options) *Storage {
	runtime.GOMAXPROCS(runtime.NumCPU())

	config, err := ParseConfig(options)
	if err != nil {
		logrus.Panicf("Cannot parse config: %s", err)
	}
	signatureService := createSignatureService(options.SignaturesPath, &config.BlacklistedStrings)

	err = os.Mkdir(options.ReportsDir, 0775)
	if err != nil && !os.IsExist(err) {
		logrus.Fatal(err)
	}
	if options.Dodjo.Url != "" && options.Dodjo.Token != "" && options.Dodjo.Product != "" {
		DefectDodjo = dodjo.New(options.Dodjo.Url, options.Dodjo.Token)
	}
	return &Storage{config, &options.Dodjo, signatureService, options.ReportsDir}
}

func (s *Storage) getAccessTokens() []AccessToken {
	return s.config.AccessTokens
}

func (s *Storage) getPatterns() []Pattern {
	return s.signatureService.getPatterns(&s.config.BlacklistedStrings)
}

func (s *Storage) getBlacklistedFilenames() []string {
	return s.config.BlacklistedFilenames
}

func (s *Storage) getBlacklistedProjectNames() []string {
	return s.config.BlacklistedProjectNames
}

func (s *Storage) getBlacklistedStrings() []string {
	return s.config.BlacklistedStrings
}

func (s *Storage) getBlacklistedExtensions() []string {
	return s.config.BlacklistedExtensions
}

func (s *Storage) getBlacklistedEntropyExtensions() []string {
	return s.config.BlacklistedEntropyExtensions
}

func (s *Storage) getBlacklistedPaths() []string {
	return s.config.BlacklistedPaths
}

func (s *Storage) getConfigPath() string {
	return s.config.Path
}

func (s *Storage) getReportsDir() string {
	return s.reportsDir
}

func (s *Storage) getDodjoProductName() string {
	return s.dodjo.Product
}
