package dirstream

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
)

const (
	DefaultChunkSize = 4096
	chunkMagicNumber = 0x9ABCDEFF
	chunkHeaderSize  = 16 // 4 bytes for magic number + 8 bytes for chunk length + 4 for CRC.
)

// writeChunks writes file data in chunks to the provided writer,
// calculating a combined CRC over the header (first 12 bytes) and the chunk data.
func writeChunks(w io.Writer, file *os.File, chunkSize int) error {
	buf := make([]byte, chunkSize)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			// Prepare the 12-byte header part.
			headerPart := make([]byte, 12)
			binary.BigEndian.PutUint32(headerPart[0:4], chunkMagicNumber)
			binary.BigEndian.PutUint64(headerPart[4:12], uint64(n))

			// Calculate CRC32 over the header part and the chunk data.
			crcValue := crc32.ChecksumIEEE(headerPart)
			crcValue = crc32.Update(crcValue, crc32.IEEETable, buf[:n])

			// Create the full header: 12 bytes of headerPart followed by 4 bytes of CRC.
			fullHeader := make([]byte, chunkHeaderSize)
			copy(fullHeader[0:12], headerPart)
			binary.BigEndian.PutUint32(fullHeader[12:16], crcValue)

			// Write the full header.
			if _, err := w.Write(fullHeader); err != nil {
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

// readChunks reads file data in chunks from the reader,
// verifies the combined CRC (over header and data), and writes the data to the file.
func readChunks(r io.Reader, file *os.File, expectedSize uint64, chunkSize int) error {
	var totalRead uint64
	for totalRead < expectedSize {
		// Read the full 16-byte header.
		fullHeader := make([]byte, chunkHeaderSize)
		n, err := io.ReadFull(r, fullHeader)
		if err != nil {
			return fmt.Errorf("error reading chunk header: expected %d bytes, got %d: %w", chunkHeaderSize, n, err)
		}

		// Split the header into its parts.
		headerPart := fullHeader[:12]
		storedCRC := binary.BigEndian.Uint32(fullHeader[12:16])

		// Validate the magic number.
		magic := binary.BigEndian.Uint32(headerPart[0:4])
		if magic != chunkMagicNumber {
			return fmt.Errorf("invalid chunk header magic: got %x, expected %x", magic, chunkMagicNumber)
		}

		chunkLength := binary.BigEndian.Uint64(headerPart[4:12])
		if chunkLength > uint64(chunkSize) {
			return fmt.Errorf("invalid chunk length %d, exceeds maximum allowed %d", chunkLength, chunkSize)
		}

		// Read the chunk data.
		chunkData := make([]byte, chunkLength)
		n, err = io.ReadFull(r, chunkData)
		if err != nil {
			return fmt.Errorf("error reading chunk data: expected %d bytes, got %d: %w", chunkLength, n, err)
		}

		// Recompute the combined CRC over the header part and the chunk data.
		crcValue := crc32.ChecksumIEEE(headerPart)
		crcValue = crc32.Update(crcValue, crc32.IEEETable, chunkData)

		// Compare the computed CRC with the stored CRC.
		if crcValue != storedCRC {
			return fmt.Errorf("CRC mismatch for chunk: expected %x, got %x", storedCRC, crcValue)
		}

		// Write the chunk data to the file.
		if _, err := file.Write(chunkData); err != nil {
			return fmt.Errorf("error writing to file: %w", err)
		}
		totalRead += chunkLength
	}
	return nil
}
