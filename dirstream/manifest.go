package dirstream

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Constants for the manifest.
const (
	manifestMagicNumber = 0x4D414E49 // Example: "MANI" in hex.
	manifestVersion     = 1
)

// ManifestEntry represents a single entry in the manifest.
type ManifestEntry struct {
	HeaderOffset uint64 // Offset where the file's header starts in the stream.
	FileSize     uint64 // File size in bytes.
	FileType     byte   // File type.
	FilePath     string // Relative file path.
}

func writeManifest(w io.Writer, entries []ManifestEntry) error {
	totalSize := 16 + 4
	for _, entry := range entries {
		totalSize += 8 + 8 + 1 + 2 + len(entry.FilePath)
	}

	buf := make([]byte, totalSize)
	offset := 0

	// Write manifest header.
	binary.BigEndian.PutUint32(buf[offset:offset+4], manifestMagicNumber)
	offset += 4
	binary.BigEndian.PutUint32(buf[offset:offset+4], manifestVersion)
	offset += 4
	binary.BigEndian.PutUint64(buf[offset:offset+8], uint64(len(entries)))
	offset += 8

	// Write each manifest entry.
	for _, entry := range entries {
		// Write HeaderOffset (8 bytes).
		binary.BigEndian.PutUint64(buf[offset:offset+8], entry.HeaderOffset)
		offset += 8

		// Write FileSize (8 bytes).
		binary.BigEndian.PutUint64(buf[offset:offset+8], entry.FileSize)
		offset += 8

		// Write FileType (1 byte).
		buf[offset] = entry.FileType
		offset++

		// Convert FilePath to bytes.
		pathBytes := []byte(entry.FilePath)
		pathLen := uint16(len(pathBytes))

		// Write FilePath length (2 bytes).
		binary.BigEndian.PutUint16(buf[offset:offset+2], pathLen)
		offset += 2

		// Write FilePath.
		copy(buf[offset:offset+len(pathBytes)], pathBytes)
		offset += len(pathBytes)
	}

	// Write trailer (4 bytes) using the same magic number.
	binary.BigEndian.PutUint32(buf[offset:offset+4], manifestMagicNumber)
	offset += 4

	// Write the complete buffer to the writer.
	_, err := w.Write(buf)
	return err
}

// readManifest reads and prints the manifest from the reader using the fixed binary layout.
func readManifest(r io.Reader) error {
	// Read the manifest header (16 bytes).
	header := make([]byte, 16)
	if _, err := io.ReadFull(r, header); err != nil {
		return fmt.Errorf("error reading manifest header: %v", err)
	}
	magic := binary.BigEndian.Uint32(header[0:4])
	if magic != manifestMagicNumber {
		return fmt.Errorf("invalid manifest magic: expected 0x%X, got 0x%X", manifestMagicNumber, magic)
	}
	version := binary.BigEndian.Uint32(header[4:8])
	if version != manifestVersion {
		return fmt.Errorf("unsupported manifest version: %d", version)
	}
	entryCount := binary.BigEndian.Uint64(header[8:16])
	fmt.Printf("Manifest contains %d entries:\n", entryCount)

	// For each entry, read the fixed part then the variable-length file path.
	for i := uint64(0); i < entryCount; i++ {
		// Fixed-size part for the entry: 8 + 8 + 1 + 2 = 19 bytes.
		entryHeader := make([]byte, 19)
		if _, err := io.ReadFull(r, entryHeader); err != nil {
			return fmt.Errorf("error reading manifest entry header: %v", err)
		}
		headerOffset := binary.BigEndian.Uint64(entryHeader[0:8])
		fileSize := binary.BigEndian.Uint64(entryHeader[8:16])
		fileType := entryHeader[16]
		pathLength := binary.BigEndian.Uint16(entryHeader[17:19])

		// Read the file path.
		pathBytes := make([]byte, pathLength)
		if _, err := io.ReadFull(r, pathBytes); err != nil {
			return fmt.Errorf("error reading file path: %v", err)
		}

		fmt.Printf("Entry %d:\n", i+1)
		fmt.Printf("  Header Offset: %d\n", headerOffset)
		fmt.Printf("  File Size: %d\n", fileSize)
		fmt.Printf("  File Type: %d\n", fileType)
		fmt.Printf("  File Path: %s\n", string(pathBytes))
	}

	// Read and validate the trailer (4 bytes).
	trailer := make([]byte, 4)
	if _, err := io.ReadFull(r, trailer); err != nil {
		return fmt.Errorf("error reading manifest trailer: %v", err)
	}
	trailerVal := binary.BigEndian.Uint32(trailer)
	if trailerVal != manifestMagicNumber {
		return fmt.Errorf("invalid manifest trailer: expected 0x%X, got 0x%X", manifestMagicNumber, trailerVal)
	}

	fmt.Println("Manifest read successfully.")
	return nil
}
