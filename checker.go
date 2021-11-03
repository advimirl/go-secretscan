package main

import (
	"os"
	"path/filepath"
	"strings"
)

type Checker struct {
	storage *Storage
}

func (c *Checker) checkFileExtBlacklisted(path string) bool {
	extension := strings.ToLower(filepath.Ext(path))

	for _, skippableExt := range c.storage.getBlacklistedExtensions() {
		if extension == skippableExt {
			return true
		}
	}

	for _, skippablePathIndicator := range c.storage.getBlacklistedPaths() {
		skippablePathIndicator = strings.Replace(skippablePathIndicator, "{sep}", string(os.PathSeparator), -1)
		if strings.Contains(path, skippablePathIndicator) {
			return true
		}
	}

	return false
}

func (c *Checker) checkFilenameBlacklisted(path string) bool {
	filename := strings.ToLower(filepath.Base(path))

	for _, skippableFilename := range c.storage.getBlacklistedFilenames() {
		if strings.Contains(filename, skippableFilename) {
			return true
		}
	}

	return false
}

func (c *Checker) checkProjectNameBlacklisted(projectName string) bool {
	pName := strings.ToLower(projectName)
	for _, skippableProject := range c.storage.getBlacklistedProjectNames() {
		if strings.Contains(pName, skippableProject) {
			return true
		}
	}
	return false
}

func (c *Checker) checkEntropyFile(filename, extension string) bool {
	if filename == "id_rsa" {
		return false
	}
	for _, skipExt := range c.storage.getBlacklistedEntropyExtensions() {
		if extension == skipExt {
			return false
		}
	}
	return true
}
