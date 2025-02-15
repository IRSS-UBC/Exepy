package common

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
)

type FileHash struct {
	RelativePath string `json:"relative_path"`
	Hash         string `json:"hash"`
}

// https://stackoverflow.com/a/40436529 CC BY-SA 4.0
func Md5SumFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func ComputeDirectoryHashes(dirPath string, ignoredDirs []string) ([]FileHash, error) {
	var fileHashes []FileHash

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// If it's a directory, check if it should be ignored.
		if info.IsDir() {
			for _, ignored := range ignoredDirs {
				if info.Name() == ignored {
					// Skip the entire directory and its contents.
					return filepath.SkipDir
				}
			}
			// Continue walking if the directory is not ignored.
			return nil
		}

		// For files, compute the relative path.
		relativePath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		// Open the file.
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Compute the MD5 hash.
		hash := md5.New()
		if _, err := io.Copy(hash, file); err != nil {
			return err
		}

		// Append the hash result.
		fileHashes = append(fileHashes, FileHash{
			RelativePath: filepath.ToSlash(relativePath),
			Hash:         hex.EncodeToString(hash.Sum(nil)),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort the file hashes by relative path to ensure a consistent order.
	sort.Slice(fileHashes, func(i, j int) bool {
		return fileHashes[i].RelativePath < fileHashes[j].RelativePath
	})

	return fileHashes, nil
}

func VerifyDirectoryHashes(dirPath string, fileHashes []FileHash) ([]string, error) {
	var mismatched []string

	for _, fh := range fileHashes {
		fullPath := filepath.Join(dirPath, fh.RelativePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// File does not exist, add to mismatched
			mismatched = append(mismatched, fh.RelativePath)
			continue
		}

		currentHash, err := Md5SumFile(fullPath)
		if err != nil {
			return nil, err
		}

		// Check if the current file's hash matches the expected hash
		if currentHash != fh.Hash {
			mismatched = append(mismatched, fh.RelativePath)
		}
	}

	return mismatched, nil
}

func HashReadSeeker(rs io.ReadSeeker) (string, error) {
	// Save the current position
	startPos, err := rs.Seek(0, io.SeekCurrent)
	if err != nil {
		return "", err
	}

	hash := md5.New()
	if _, err := io.Copy(hash, rs); err != nil {
		return "", err
	}

	// Restore the position
	_, err = rs.Seek(startPos, io.SeekStart)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
