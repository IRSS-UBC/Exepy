package common

import (
	"bytes"
	"context"
	"fmt"
	"github.com/mholt/archiver/v4"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func getFormat() archiver.CompressedArchive {
	format := archiver.CompressedArchive{
		Compression: archiver.Bz2{},
		Archival:    archiver.Tar{},
	}
	return format
}

func CompressDirToStream(directoryPath string, ignoredDirs []string) (io.ReadSeeker, error) {
	// Get the list of files and directories in the specified folder
	FromDiskOptions := &archiver.FromDiskOptions{
		FollowSymlinks:  false,
		ClearAttributes: true,
	}

	// map the files to the archive
	pathMap, err := mapFilesAndDirectories(directoryPath, ignoredDirs)
	if err != nil {
		return nil, err
	}

	// Create a new zip archive
	files, err := archiver.FilesFromDisk(FromDiskOptions, pathMap)
	if err != nil {
		return nil, err
	}

	// create a buffer to hold the compressed data
	buf := new(bytes.Buffer)

	format := getFormat()

	// create the archive
	err = format.Archive(context.Background(), buf, files)
	if err != nil {
		return nil, err
	}

	// convert the buffer to an io.ReadSeeker
	readSeeker := bytes.NewReader(buf.Bytes())

	return readSeeker, nil
}

func DecompressIOStream(IOReader io.Reader, outputDir string) error {

	format := getFormat()

	handler := func(ctx context.Context, archivedFile archiver.File) error {

		outPath := filepath.Join(outputDir, archivedFile.NameInArchive)

		if archivedFile.FileInfo.IsDir() {
			err := os.MkdirAll(outPath, os.ModePerm)
			if err != nil {
				return err
			}

			return nil
		} else {
			dir := filepath.Dir(outPath)
			err := os.MkdirAll(dir, os.ModePerm)

			if err != nil {
				return err
			}
		}

		// Create the outputFileStream
		outputFileStream, err := os.Create(outPath)
		if err != nil {
			return err
		}

		defer outputFileStream.Close()

		archivedFileStream, err := archivedFile.Open()
		if err != nil {
			return err
		}
		defer archivedFileStream.Close()

		// Write the outputFileStream
		_, err = io.Copy(outputFileStream, archivedFileStream)

		if err != nil {
			return err
		}

		return nil
	}

	ctx := context.Background()

	err := format.Extract(ctx, IOReader, nil, handler)
	if err != nil {
		return err
	}

	return nil
}

func mapFilesAndDirectories(directoryPath string, ignoredDirs []string) (map[string]string, error) {
	// Initialize a map to store file names and their corresponding archive paths.
	fileMap := make(map[string]string)
	pathSeparator := string(os.PathSeparator)

	err := filepath.WalkDir(directoryPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute the relative path from the base directory.
		relativePath, err := filepath.Rel(directoryPath, path)
		if err != nil {
			return err
		}

		// Skip the root directory.
		if relativePath == "." {
			return nil
		}

		// Check if any segment of the relative path matches an ignored directory.
		segments := strings.Split(relativePath, string(os.PathSeparator))
		for _, segment := range segments {
			for _, ignored := range ignoredDirs {
				if segment == ignored {
					if d.IsDir() {
						// Skip the directory and its contents.

						fmt.Println("Skipping directory: ", path)

						return filepath.SkipDir
					}
					// For files within an ignored directory, simply do not add them.
					return nil
				}
			}
		}

		if d.IsDir() {
			// For directories, check if the directory is empty.
			isEmpty, err := isDirEmpty(path)
			if err != nil {
				return err
			}

			// If the directory is empty, add it to the map with a trailing separator.
			if isEmpty {
				// The key uses the OS-specific separator,
				// while the archive path uses forward slashes.
				fileMap[path] = filepath.ToSlash(relativePath + pathSeparator)
			}
		} else {
			// For files, simply map the path to its relative path.
			fileMap[path] = filepath.ToSlash(relativePath)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return fileMap, nil
}

func isDirEmpty(dirPath string) (bool, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return false, err
	}
	defer dir.Close()

	_, err = dir.Readdirnames(1)
	if err == nil {
		// Directory is not empty
		return false, nil
	}

	// Directory is empty or an error occurred
	return true, nil
}
