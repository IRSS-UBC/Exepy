package dirstream

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Decoder decodes an encoded stream back into files, directories, and symlinks.
type Decoder struct {
	destPath   string
	strictMode bool // If true, decoding stops on minor errors.
	chunkSize  int
}

// NewDecoder creates a new Decoder with an option for strict mode.
func NewDecoder(destPath string, strictMode bool, chunkSize int) *Decoder {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	return &Decoder{destPath: destPath, strictMode: strictMode, chunkSize: chunkSize}
}

// recover scans the stream byte-by-byte until the magic number is found.
// If the underlying reader supports seeking, it rewinds to re-read the full chunk header.
func (d *Decoder) recover(r io.Reader) error {
	buf := make([]byte, 1)
	magicBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(magicBytes, chunkMagicNumber)

	for {
		_, err := r.Read(buf)
		if err != nil {
			return err
		}
		// Shift the bytes left and append the new byte.
		copy(magicBytes, magicBytes[1:])
		magicBytes[3] = buf[0]
		if binary.BigEndian.Uint32(magicBytes) == chunkMagicNumber {
			if seeker, ok := r.(io.Seeker); ok {
				// Rewind by (chunkHeaderSize - 4) bytes so the full header can be re-read.
				if _, err := seeker.Seek(-int64(chunkHeaderSize-4), io.SeekCurrent); err != nil {
					return err
				}
			} else {
				return errors.New("stream does not support seeking, cannot recover")
			}
			return nil
		}
	}
}
func (d *Decoder) Decode(r io.Reader) error {
	bufferedReader := bufio.NewReader(r)

	for {
		// Check if the next file header is available or if it's a manifest.

		magicBuf, err := bufferedReader.Peek(4)
		if err == io.EOF {
			// No more data in the stream; stop decoding.
			return nil
		}
		if err != nil {
			return fmt.Errorf("Decode: error peeking magic number: %v", err)
		}

		fmt.Printf("Magic number: % x\n", magicBuf)
		magic := binary.BigEndian.Uint32(magicBuf)

		if magic == manifestMagicNumber {

			println("End of stream detected. Exiting...")

			// Read and process the manifest.
			entries, err := readManifest(bufferedReader)
			if err != nil {
				return fmt.Errorf("Decode: error reading manifest: %v", err)
			}

			for _, entry := range entries {
				fmt.Printf("Entry: %v\n", entry)
			}

			break // Stop decoding after the manifest.
		}

		// Read file header
		fh, err := readHeader(bufferedReader)
		if err == io.EOF {
			break // End of stream.
		}

		if err != nil {
			return fmt.Errorf("Decode: error reading header: %v", err)
		}

		fullPath := filepath.Join(d.destPath, fh.FilePath)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("Decode: error creating directory %s: %v", dir, err)
		}

		switch fh.FileType {
		case fileTypeDirectory:
			if err := os.MkdirAll(fullPath, os.FileMode(fh.FileMode)); err != nil {
				return fmt.Errorf("Decode: error creating directory %s: %v", fullPath, err)
			}
			fmt.Printf("Decoded directory: %s\n", fullPath)
			continue
		case fileTypeSymlink:
			os.Remove(fullPath)
			if err := os.Symlink(fh.LinkTarget, fullPath); err != nil {
				return fmt.Errorf("Decode: error creating symlink %s -> %s: %v", fullPath, fh.LinkTarget, err)
			}
			fmt.Printf("Decoded symlink: %s -> %s\n", fullPath, fh.LinkTarget)
			continue
		case fileTypeRegular:
			file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(fh.FileMode))
			if err != nil {
				return fmt.Errorf("Decode: error opening file %s: %v", fullPath, err)
			}

			if err := readChunks(bufferedReader, file, fh.FileSize, d.chunkSize); err != nil {
				file.Close()
				return fmt.Errorf("Decode: error reading chunks for file %s: %v", fh.FilePath, err)
			}
			file.Close()
			fmt.Printf("Decoded file: %s\n", fullPath)

			file.Close()
			fmt.Printf("Decoded file: %s\n", fullPath)
			continue
		default:
			return fmt.Errorf("Decode: unknown file type for %s", fh.FilePath)
		}

	}

	return nil
}
