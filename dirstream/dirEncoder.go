package dirstream

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Encoder struct {
	rootPath  string
	chunkSize int
}

func NewEncoder(rootPath string, chunkSize int) *Encoder {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	return &Encoder{rootPath: rootPath, chunkSize: chunkSize}
}

func (e *Encoder) Encode(fileList []string) (io.Reader, error) {
	r, w := io.Pipe()
	cw := &CountingWriter{w: w}
	bufferedWriter := bufio.NewWriter(cw)

	var manifestEntries []ManifestEntry

	go func() {
		defer func() {
			bufferedWriter.Flush()
			w.Close()
		}()

		for _, relPath := range fileList {
			fullPath := filepath.Join(e.rootPath, relPath)

			info, err := os.Lstat(fullPath)
			if err != nil {
				w.CloseWithError(err)
				return
			}

			var fh fileHeader
			fh.Version = headerVersion
			fh.FilePath = relPath
			fh.ModTime = info.ModTime().Unix()
			fh.FileMode = uint32(info.Mode())

			if info.IsDir() {
				fh.FileSize = 0
				fh.FileType = fileTypeDirectory
				fh.LinkTarget = ""
			} else if info.Mode()&os.ModeSymlink != 0 {
				linkTarget, err := os.Readlink(fullPath)
				if err != nil {
					err := w.CloseWithError(err)
					if err != nil {
						return
					}
					return
				}
				fh.FileSize = 0
				fh.FileType = fileTypeSymlink
				fh.LinkTarget = linkTarget
			} else if info.Mode().IsRegular() {
				fh.FileSize = uint64(info.Size())
				fh.FileType = fileTypeRegular
				fh.LinkTarget = ""
			} else {
				continue
			}

			if err := bufferedWriter.Flush(); err != nil {
				w.CloseWithError(err)
				return
			}
			offset := cw.Count

			if err := writeHeader(bufferedWriter, fh); err != nil {
				w.CloseWithError(err)
				return
			}

			manifestEntries = append(manifestEntries, ManifestEntry{
				HeaderOffset: offset,
				FileSize:     fh.FileSize,
				FileType:     fh.FileType,
				FilePath:     fh.FilePath,
			})

			if fh.FileType == fileTypeRegular {
				file, err := os.Open(fullPath)
				if err != nil {
					w.CloseWithError(err)
					return
				}

				if err := writeChunks(bufferedWriter, file, e.chunkSize); err != nil {
					file.Close()
					w.CloseWithError(err)
					return
				}
				file.Close()
				fmt.Printf("Encoded file: %s\n", relPath)
			} else if fh.FileType == fileTypeDirectory {
				fmt.Printf("Encoded directory: %s\n", relPath)
			} else if fh.FileType == fileTypeSymlink {
				fmt.Printf("Encoded symlink: %s -> %s\n", relPath, fh.LinkTarget)
			}
		}

		if err := bufferedWriter.Flush(); err != nil {
			w.CloseWithError(err)
			return
		}

		if err := writeManifest(bufferedWriter, manifestEntries); err != nil {
			w.CloseWithError(err)
			return
		}
	}()

	return r, nil
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
		files = append(files, relPath)
		return nil
	})

	return files, err
}
