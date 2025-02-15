package dirstream

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// -----------------------------------------------------------------------------
// Encoder
// -----------------------------------------------------------------------------

// Encoder encodes a file system tree into a single io.Reader stream.
type Encoder struct {
	rootPath  string
	chunkSize int
}

// NewEncoder creates a new Encoder with a configurable chunk size.
func NewEncoder(rootPath string, chunkSize int) *Encoder {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	return &Encoder{rootPath: rootPath, chunkSize: chunkSize}
}

// Encode walks the directory tree and writes file headers and data to a stream.
func (e *Encoder) Encode() (io.Reader, error) {
	r, w := io.Pipe()
	// Wrap the writer with a counting writer then a buffered writer.
	cw := &CountingWriter{w: w}
	bufferedWriter := bufio.NewWriter(cw)

	// In-memory manifest table.
	var manifestEntries []ManifestEntry

	go func() {
		// Ensure the buffered writer is flushed and the pipe is closed.
		defer func() {
			// Flush any remaining buffered data.
			bufferedWriter.Flush()
			w.Close()
		}()

		err := filepath.WalkDir(e.rootPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Get the relative path.
			relPath, err := filepath.Rel(e.rootPath, path)
			if err != nil {
				return err
			}

			// Retrieve file info.
			info, err := d.Info()
			if err != nil {
				return err
			}

			// Prepare the file header.
			var fh fileHeader
			fh.Version = headerVersion
			fh.FilePath = relPath
			fh.ModTime = info.ModTime().Unix()
			fh.FileMode = uint32(info.Mode())

			// Process directories.
			if d.IsDir() {
				fh.FileSize = 0
				fh.FileType = FileTypeDirectory
				fh.LinkTarget = ""
			} else if d.Type()&os.ModeSymlink != 0 { // Process symlinks.
				// Use Lstat to avoid following the symlink.
				info, err = os.Lstat(path)
				if err != nil {
					return err
				}
				linkTarget, err := os.Readlink(path)
				if err != nil {
					return err
				}
				fh.FileSize = 0
				fh.FileType = FileTypeSymlink
				fh.LinkTarget = linkTarget
			} else if info.Mode().IsRegular() { // Process regular files.
				fh.FileSize = uint64(info.Size())
				fh.FileType = FileTypeRegular
				fh.LinkTarget = ""
			} else {
				// Skip non-regular files.
				return nil
			}

			// Flush the buffer to ensure cw.Count is up-to-date.
			if err := bufferedWriter.Flush(); err != nil {
				return err
			}
			// Record the current offset as the header start.
			offset := cw.Count

			// Write the header.
			if err := writeHeader(bufferedWriter, fh); err != nil {
				return err
			}
			// Add an entry to the manifest.
			manifestEntries = append(manifestEntries, ManifestEntry{
				HeaderOffset: offset,
				FileSize:     fh.FileSize,
				FileType:     fh.FileType,
				FilePath:     fh.FilePath,
			})

			// For regular files, write file data in chunks.
			if fh.FileType == FileTypeRegular {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()

				buf := make([]byte, e.chunkSize)
				for {
					n, err := file.Read(buf)
					if n > 0 {
						// Write chunk header: 4 bytes for magic number and 8 bytes for chunk length.
						chunkHeader := make([]byte, chunkHeaderSize)
						binary.BigEndian.PutUint32(chunkHeader[0:4], chunkMagicNumber)
						binary.BigEndian.PutUint64(chunkHeader[4:12], uint64(n))
						if _, err := bufferedWriter.Write(chunkHeader); err != nil {
							return err
						}
						if _, err := bufferedWriter.Write(buf[:n]); err != nil {
							return err
						}
					}
					if err == io.EOF {
						break
					}
					if err != nil {
						return err
					}
				}
				fmt.Printf("Encoded file: %s\n", relPath)
			} else if fh.FileType == FileTypeDirectory {
				fmt.Printf("Encoded directory: %s\n", relPath)
			} else if fh.FileType == FileTypeSymlink {
				fmt.Printf("Encoded symlink: %s -> %s\n", relPath, fh.LinkTarget)
			}

			return nil
		})

		bufferedWriter.Flush()

		// Write the manifest.
		//if err := WriteManifest(bufferedWriter, manifestEntries); err != nil {
		//	w.CloseWithError(err)
		//}

		if err != nil {
			w.CloseWithError(err)
		}
	}()

	return r, nil
}
