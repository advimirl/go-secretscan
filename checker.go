package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

func checkFileExtBlacklisted(ctx context.Context, path string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	storage := getContextStorage(ctx)
	for _, skippableExt := range storage.getBlacklistedExtensions() {
		if extension == skippableExt {
			return true
		}
	}

	for _, skippablePathIndicator := range storage.getBlacklistedPaths() {
		skippablePathIndicator = strings.Replace(skippablePathIndicator, "{sep}", string(os.PathSeparator), -1)
		if strings.Contains(path, skippablePathIndicator) {
			return true
		}
	}

	return false
}

func checkFilenameBlacklisted(ctx context.Context, path string) bool {
	filename := strings.ToLower(filepath.Base(path))
	storage := getContextStorage(ctx)
	for _, skippableFilename := range storage.getBlacklistedFilenames() {
		if strings.Contains(filename, skippableFilename) {
			return true
		}
	}

	return false
}

func checkProjectNameBlacklisted(ctx context.Context, projectName string) bool {
	pName := strings.ToLower(projectName)
	storage := getContextStorage(ctx)
	for _, skippableProject := range storage.getBlacklistedProjectNames() {
		if strings.Contains(pName, skippableProject) {
			return true
		}
	}
	return false
}

func checkEntropyFile(ctx context.Context, filename, extension string) bool {
	if filename == "id_rsa" {
		return false
	}
	storage := getContextStorage(ctx)
	for _, skipExt := range storage.getBlacklistedEntropyExtensions() {
		if extension == skipExt {
			return false
		}
	}
	return true
}
