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
	// Wrap the writer with a buffered writer.
	bufferedWriter := bufio.NewWriter(w)

	go func() {
		// Ensure the buffered writer is flushed and the pipe is closed.
		defer func() {
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

			// Process directories.
			if d.IsDir() {
				fh := fileHeader{
					Version:    headerVersion,
					FilePath:   relPath,
					FileSize:   0,
					FileMode:   uint32(info.Mode()),
					ModTime:    info.ModTime().Unix(),
					FileType:   FileTypeDirectory,
					LinkTarget: "",
				}
				if err := writeHeader(bufferedWriter, fh); err != nil {
					return err
				}
				fmt.Printf("Encoded directory: %s\n", relPath)
				return nil
			}

			// Process symlinks.
			if d.Type()&os.ModeSymlink != 0 {
				// Use Lstat to avoid following the symlink.
				info, err = os.Lstat(path)
				if err != nil {
					return err
				}
				linkTarget, err := os.Readlink(path)
				if err != nil {
					return err
				}
				fh := fileHeader{
					Version:    headerVersion,
					FilePath:   relPath,
					FileSize:   0,
					FileMode:   uint32(info.Mode()),
					ModTime:    info.ModTime().Unix(),
					FileType:   FileTypeSymlink,
					LinkTarget: linkTarget,
				}
				if err := writeHeader(bufferedWriter, fh); err != nil {
					return err
				}
				fmt.Printf("Encoded symlink: %s -> %s\n", relPath, linkTarget)
				return nil
			}

			// Process regular files.
			if !info.Mode().IsRegular() {
				// Skip non-regular files.
				return nil
			}

			fh := fileHeader{
				Version:    headerVersion,
				FilePath:   relPath,
				FileSize:   uint64(info.Size()),
				FileMode:   uint32(info.Mode()),
				ModTime:    info.ModTime().Unix(),
				FileType:   FileTypeRegular,
				LinkTarget: "",
			}
			if err := writeHeader(bufferedWriter, fh); err != nil {
				return err
			}

			// Open the file for reading.
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
					binary.BigEndian.PutUint32(chunkHeader[0:4], magicNumber)
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
			return nil
		})

		if err != nil {
			w.CloseWithError(err)
		}
	}()

	return r, nil
}
