package dirstream

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	DefaultChunkSize = 4096
	chunkMagicNumber = 0x9ABCDEFF
	chunkHeaderSize  = 12 // 4 bytes for magic number + 8 bytes for chunk length.
)

// writeChunks writes file data in chunks to the provided writer.
// The function reads from the open file and writes each chunk with a header.
func writeChunks(w io.Writer, file *os.File, chunkSize int) error {
	buf := make([]byte, chunkSize)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			// Prepare and write the chunk header.
			chunkHeader := make([]byte, chunkHeaderSize)
			binary.BigEndian.PutUint32(chunkHeader[0:4], chunkMagicNumber)
			binary.BigEndian.PutUint64(chunkHeader[4:12], uint64(n))
			if _, err := w.Write(chunkHeader); err != nil {
				return err
			}
			// Write the chunk data.
			if _, err := w.Write(buf[:n]); err != nil {
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
	return nil
}

// readChunks reads file data in chunks from the reader and writes it to the given file.
// It continues until the expectedSize of data is read.
func readChunks(r io.Reader, file *os.File, expectedSize uint64, chunkSize int) error {
	var totalRead uint64
	for totalRead < expectedSize {
		chunkHeader := make([]byte, chunkHeaderSize)
		n, err := io.ReadFull(r, chunkHeader)
		if err != nil {
			return fmt.Errorf("error reading chunk header: expected %d bytes, got %d: %w", chunkHeaderSize, n, err)
		}

		readMagic := binary.BigEndian.Uint32(chunkHeader[0:4])
		if readMagic != chunkMagicNumber {
			return fmt.Errorf("invalid chunk header magic: got %x, expected %x", readMagic, chunkMagicNumber)
		}

		chunkLength := binary.BigEndian.Uint64(chunkHeader[4:12])
		if chunkLength > uint64(chunkSize) {
			return fmt.Errorf("invalid chunk length %d, exceeds maximum allowed %d", chunkLength, chunkSize)
		}

		chunkData := make([]byte, chunkLength)
		n, err = io.ReadFull(r, chunkData)
		if err != nil {
			return fmt.Errorf("error reading chunk data: expected %d bytes, got %d: %w", chunkLength, n, err)
		}

		if _, err := file.Write(chunkData); err != nil {
			return fmt.Errorf("error writing to file: %w", err)
		}
		totalRead += chunkLength
	}
	return nil
}
