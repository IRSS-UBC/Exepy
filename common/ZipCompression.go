package common

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ExtractZip(zipFile, extractDir string, skipLevels int) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, file := range r.File {
		components := strings.Split(file.Name, "/")

		if len(components) > skipLevels {
			relativePath := strings.Join(components[skipLevels:], "/")
			path := filepath.Join(extractDir, relativePath)

			if file.FileInfo().IsDir() {
				if err := os.MkdirAll(path, os.ModePerm); err != nil {
					return err
				}
				continue
			}

			if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
				return err
			}

			if err := extractFile(file, path); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractFile(file *zip.File, path string) error {
	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()

	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	if _, err := io.Copy(outFile, rc); err != nil {
		return err
	}
	return nil
}
