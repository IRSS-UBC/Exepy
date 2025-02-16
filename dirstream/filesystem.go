package dirstream

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func sanitizePath(destPath, filePath string) (string, error) {
	destPath = filepath.Clean(destPath) + string(filepath.Separator) // Ensure trailing slash for directory
	joinedPath := filepath.Join(destPath, filePath)
	cleanPath := filepath.Clean(joinedPath)

	if !strings.HasPrefix(cleanPath, destPath) {
		return "", fmt.Errorf("invalid path: %s", filePath) // Prevent directory traversal
	}
	if strings.Contains(cleanPath, "..") { // Additional check
		return "", fmt.Errorf("invalid path contains '..': %s", filePath)
	}
	if filepath.IsAbs(filePath) { // Check absolute file paths
		return "", fmt.Errorf("invalid path: absolute paths not allowed: %s", filePath)
	}
	return cleanPath, nil
}

func BuildRelativeFileList(rootPath string, excludes []string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		for _, exclude := range excludes {
			if strings.Contains(path, exclude) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}

		// ADD THIS CHECK:
		if relPath == "." {
			return nil // Skip the root path itself
		}

		files = append(files, relPath)
		return nil
	})

	return files, err
}
