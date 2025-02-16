package dirstream

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
)

const (
	manifestMagicNumber = 0x4D414E49 // 'MANI'
	manifestVersion     = 1
)

type ManifestEntry struct {
	HeaderOffset uint64 // Offset where the file's header starts in the stream.
	FileSize     uint64 // File size in bytes.
	FileType     byte   // File type.
	FilePath     string // Relative file path.
}

// writeManifest writes the manifest
func writeManifest(w io.Writer, entries []ManifestEntry) error {
	// Header (16 bytes) + Trailer (4 bytes) + CRC (4 bytes)
	totalSize := 16 + 4 + 4
	// For each entry: fixed part (8+8+1+2 = 19 bytes) + file path length.
	for _, entry := range entries {
		totalSize += 19 + len(entry.FilePath)
	}

	buf := make([]byte, totalSize)
	offset := 0

	binary.BigEndian.PutUint32(buf[offset:offset+4], manifestMagicNumber)
	offset += 4
	binary.BigEndian.PutUint32(buf[offset:offset+4], manifestVersion)
	offset += 4
	binary.BigEndian.PutUint64(buf[offset:offset+8], uint64(len(entries)))
	offset += 8

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

	// Compute CRC32 over all bytes written so far.
	crcValue := crc32.ChecksumIEEE(buf[:offset])
	binary.BigEndian.PutUint32(buf[offset:offset+4], crcValue)
	offset += 4

	// Write the complete buffer to the writer.
	_, err := w.Write(buf)
	return err
}

// readManifest reads the manifest from the reader in a streaming fashion.
// It reads each part sequentially while updating a CRC32 hash, and then verifies the stored CRC.
func readManifest(r io.Reader) ([]ManifestEntry, error) {
	h := crc32.NewIEEE()
	tr := io.TeeReader(r, h)

	// Read the manifest header (16 bytes):
	//   - 4 bytes magic number
	//   - 4 bytes version
	//   - 8 bytes entry count
	header := make([]byte, 16)
	if _, err := io.ReadFull(tr, header); err != nil {
		return nil, fmt.Errorf("error reading manifest header: %w", err)
	}

	magic := binary.BigEndian.Uint32(header[0:4])
	version := binary.BigEndian.Uint32(header[4:8])
	entryCount := binary.BigEndian.Uint64(header[8:16])
	if magic != manifestMagicNumber {
		return nil, fmt.Errorf("invalid manifest magic: expected 0x%X, got 0x%X", manifestMagicNumber, magic)
	}
	if version != manifestVersion {
		return nil, fmt.Errorf("unsupported manifest version: %d", version)
	}

	// Prepare a slice to hold the entries.
	entries := make([]ManifestEntry, 0, entryCount)

	// Process each manifest entry.
	for i := uint64(0); i < entryCount; i++ {
		// Read the fixed part (19 bytes).
		fixedPart := make([]byte, 19)
		if _, err := io.ReadFull(tr, fixedPart); err != nil {
			return nil, fmt.Errorf("error reading fixed part for manifest entry %d: %w", i, err)
		}

		headerOffset := binary.BigEndian.Uint64(fixedPart[0:8])
		fileSize := binary.BigEndian.Uint64(fixedPart[8:16])
		fileType := fixedPart[16]
		pathLen := binary.BigEndian.Uint16(fixedPart[17:19])

		// Read the variable-length FilePath.
		pathBytes := make([]byte, pathLen)
		if _, err := io.ReadFull(tr, pathBytes); err != nil {
			return nil, fmt.Errorf("error reading file path for manifest entry %d: %w", i, err)
		}
		filePath := string(pathBytes)

		entries = append(entries, ManifestEntry{
			HeaderOffset: headerOffset,
			FileSize:     fileSize,
			FileType:     fileType,
			FilePath:     filePath,
		})
	}

	// Read the trailer (4 bytes), which should match the magic number.
	trailer := make([]byte, 4)
	if _, err := io.ReadFull(tr, trailer); err != nil {
		return nil, fmt.Errorf("error reading manifest trailer: %w", err)
	}
	trailerValue := binary.BigEndian.Uint32(trailer)
	if trailerValue != manifestMagicNumber {
		return nil, fmt.Errorf("invalid manifest trailer: expected 0x%X, got 0x%X", manifestMagicNumber, trailerValue)
	}

	// Now read the final CRC (4 bytes) directly from the original reader.
	// We do this directly to avoid including these bytes in the CRC computation.
	crcBytes := make([]byte, 4)
	if _, err := io.ReadFull(r, crcBytes); err != nil {
		return nil, fmt.Errorf("error reading manifest CRC: %w", err)
	}
	storedCrc := binary.BigEndian.Uint32(crcBytes)
	computedCrc := h.Sum32()
	if storedCrc != computedCrc {
		return nil, fmt.Errorf("manifest CRC mismatch: expected 0x%X, got 0x%X", storedCrc, computedCrc)
	}

	fmt.Println("Manifest read successfully.")
	return entries, nil
}
