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

// -----------------------------------------------------------------------------
// Decoder
// -----------------------------------------------------------------------------

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
	binary.BigEndian.PutUint32(magicBytes, magicNumber)

	for {
		_, err := r.Read(buf)
		if err != nil {
			return err
		}
		// Shift the bytes left and append the new byte.
		copy(magicBytes, magicBytes[1:])
		magicBytes[3] = buf[0]
		if binary.BigEndian.Uint32(magicBytes) == magicNumber {
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

// Decode reads from the encoded stream and recreates the files, directories, and symlinks.
func (d *Decoder) Decode(r io.Reader) error {
	bufferedReader := bufio.NewReader(r)

	for {
		fh, err := readHeader(bufferedReader)
		if err == io.EOF {
			break // End of stream.
		}
		if err != nil {
			return err
		}

		fullPath := filepath.Join(d.destPath, fh.FilePath)
		// Ensure the parent directory exists.
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		switch fh.FileType {
		case FileTypeDirectory:
			if err := os.MkdirAll(fullPath, os.FileMode(fh.FileMode)); err != nil {
				return err
			}
			fmt.Printf("Decoded directory: %s\n", fullPath)
			continue
		case FileTypeSymlink:
			// Remove any existing file or link.
			os.Remove(fullPath)
			if err := os.Symlink(fh.LinkTarget, fullPath); err != nil {
				return err
			}
			fmt.Printf("Decoded symlink: %s -> %s\n", fullPath, fh.LinkTarget)
			continue
		case FileTypeRegular:
			// Proceed to decode file contents.
		default:
			return fmt.Errorf("unknown file type for %s", fh.FilePath)
		}

		// Create or truncate the file.
		file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(fh.FileMode))
		if err != nil {
			return err
		}

		var totalRead uint64 = 0
		for totalRead < fh.FileSize {
			chunkHeader := make([]byte, chunkHeaderSize)
			if _, err := io.ReadFull(bufferedReader, chunkHeader); err != nil {
				file.Close()
				fmt.Fprintf(os.Stderr, "Error reading chunk header for %s: %v\n", fh.FilePath, err)
				if recErr := d.recover(bufferedReader); recErr != nil {
					if recErr == io.EOF {
						return nil
					}
					if d.strictMode {
						return recErr
					}
					fmt.Fprintf(os.Stderr, "Recovery failed for %s: %v\n", fh.FilePath, recErr)
					break
				}
				break
			}

			readMagic := binary.BigEndian.Uint32(chunkHeader[0:4])
			if readMagic != magicNumber {
				file.Close()
				fmt.Fprintf(os.Stderr, "Invalid chunk header magic for %s, attempting recovery...\n", fh.FilePath)
				if recErr := d.recover(bufferedReader); recErr != nil {
					if recErr == io.EOF {
						return nil
					}
					if d.strictMode {
						return recErr
					}
					fmt.Fprintf(os.Stderr, "Recovery failed for %s: %v\n", fh.FilePath, recErr)
					break
				}
				break
			}

			chunkLength := binary.BigEndian.Uint64(chunkHeader[4:12])
			if chunkLength > uint64(d.chunkSize) {
				file.Close()
				return fmt.Errorf("invalid chunk length %d for file %s", chunkLength, fh.FilePath)
			}

			chunkData := make([]byte, chunkLength)
			if _, err := io.ReadFull(bufferedReader, chunkData); err != nil {
				file.Close()
				return fmt.Errorf("error reading chunk data for %s: %w", fh.FilePath, err)
			}

			if _, err := file.Write(chunkData); err != nil {
				file.Close()
				return fmt.Errorf("error writing to file %s: %w", fh.FilePath, err)
			}
			totalRead += chunkLength
		}
		file.Close()
		fmt.Printf("Decoded file: %s\n", fullPath)
	}

	return nil
}
