package dirstream

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
)

const (
	manifestMagicNumber = 0x4D414E49 // 'MANI'
	manifestVersion     = 1
)

// ManifestEntry represents a single entry in the manifest.
type ManifestEntry struct {
	HeaderOffset uint64 // Offset where the file's header starts in the stream.
	FileSize     uint64 // File size in bytes.
	FileType     byte   // File type.
	FilePath     string // Relative file path.
}

// writeManifest writes the manifest with the following layout:
//   - Manifest header: 16 bytes (4 bytes magic, 4 bytes version, 8 bytes entry count)
//   - For each entry: 8 bytes HeaderOffset, 8 bytes FileSize, 1 byte FileType,
//     2 bytes FilePath length, variable-length FilePath
//   - Trailer: 4 bytes (same magic number)
//   - CRC: 4 bytes (CRC32 computed over everything above)
func writeManifest(w io.Writer, entries []ManifestEntry) error {
	// Calculate total size.
	// Header (16 bytes) + Trailer (4 bytes) + CRC (4 bytes)
	totalSize := 16 + 4 + 4
	// For each entry: fixed part (8+8+1+2 = 19 bytes) + file path length.
	for _, entry := range entries {
		totalSize += 19 + len(entry.FilePath)
	}

	buf := make([]byte, totalSize)
	offset := 0

	// Write manifest header (16 bytes).
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

	// Compute CRC32 over all bytes written so far.
	crcValue := crc32.ChecksumIEEE(buf[:offset])
	binary.BigEndian.PutUint32(buf[offset:offset+4], crcValue)
	offset += 4

	// Write the complete buffer to the writer.
	_, err := w.Write(buf)
	return err
}

// readManifest reads the entire manifest from the reader, verifies the CRC,
// and parses the manifest entries.
func readManifest(r io.Reader) ([]ManifestEntry, error) {
	// Read the entire manifest into memory.
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading manifest: %v", err)
	}

	// The manifest must be at least header (16) + trailer (4) + CRC (4) = 24 bytes.
	if len(buf) < 24 {
		return nil, fmt.Errorf("manifest too short: %d bytes", len(buf))
	}

	// The last 4 bytes are the CRC.
	crcStored := binary.BigEndian.Uint32(buf[len(buf)-4:])
	crcCalculated := crc32.ChecksumIEEE(buf[:len(buf)-4])
	if crcStored != crcCalculated {
		return nil, fmt.Errorf("manifest CRC mismatch: expected 0x%X, got 0x%X", crcStored, crcCalculated)
	}

	// Parse the manifest (excluding the final 4-byte CRC).
	offset := 0

	// Manifest header (16 bytes).
	magic := binary.BigEndian.Uint32(buf[offset : offset+4])
	offset += 4
	if magic != manifestMagicNumber {
		return nil, fmt.Errorf("invalid manifest magic: expected 0x%X, got 0x%X", manifestMagicNumber, magic)
	}

	version := binary.BigEndian.Uint32(buf[offset : offset+4])
	offset += 4
	if version != manifestVersion {
		return nil, fmt.Errorf("unsupported manifest version: %d", version)
	}

	entryCount := binary.BigEndian.Uint64(buf[offset : offset+8])
	offset += 8

	entries := make([]ManifestEntry, entryCount)

	// Parse each manifest entry.
	for i := uint64(0); i < entryCount; i++ {
		// Each entry's fixed part is 8+8+1+2 = 19 bytes.
		if offset+19 > len(buf)-4 {
			return nil, fmt.Errorf("manifest entry %d incomplete", i)
		}

		headerOffset := binary.BigEndian.Uint64(buf[offset : offset+8])
		offset += 8
		fileSize := binary.BigEndian.Uint64(buf[offset : offset+8])
		offset += 8
		fileType := buf[offset]
		offset++
		pathLen := binary.BigEndian.Uint16(buf[offset : offset+2])
		offset += 2

		// Read the file path.
		if offset+int(pathLen) > len(buf)-4 {
			return nil, fmt.Errorf("manifest entry %d file path incomplete", i)
		}
		filePath := string(buf[offset : offset+int(pathLen)])
		offset += int(pathLen)

		entries[i] = ManifestEntry{
			HeaderOffset: headerOffset,
			FileSize:     fileSize,
			FileType:     fileType,
			FilePath:     filePath,
		}
	}

	// Read and validate the trailer (4 bytes).
	if offset+4 > len(buf)-4 {
		return nil, fmt.Errorf("manifest trailer missing")
	}
	trailer := binary.BigEndian.Uint32(buf[offset : offset+4])
	offset += 4
	if trailer != manifestMagicNumber {
		return nil, fmt.Errorf("invalid manifest trailer: expected 0x%X, got 0x%X", manifestMagicNumber, trailer)
	}

	// Ensure we've consumed all data except the final CRC.
	if offset != len(buf)-4 {
		return nil, fmt.Errorf("unexpected extra data in manifest")
	}

	fmt.Println("Manifest read successfully.")
	return entries, nil
}
