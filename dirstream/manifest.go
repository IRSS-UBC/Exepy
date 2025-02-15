package dirstream

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	manifestMagicNumber = 0xABCD1234
	manifestVersion     = 1
)

// ManifestEntry represents a single entry in the manifest.
type ManifestEntry struct {
	HeaderOffset uint64 // Offset where the file's header starts in the stream.
	FileSize     uint64 // File size in bytes.
	FileType     byte   // File type.
	FilePath     string // Relative file path.
}

// WriteManifest writes the manifest to the writer.
func WriteManifest(w io.Writer, entries []ManifestEntry) error {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, manifestMagicNumber); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.BigEndian, uint32(manifestVersion)); err != nil {
		return err
	}
	entryCount := uint64(len(entries))
	if err := binary.Write(buf, binary.BigEndian, entryCount); err != nil {
		return err
	}

	// Write each manifest entry.
	for _, entry := range entries {
		// Write header offset.
		if err := binary.Write(buf, binary.BigEndian, entry.HeaderOffset); err != nil {
			return err
		}
		// Write file size.
		if err := binary.Write(buf, binary.BigEndian, entry.FileSize); err != nil {
			return err
		}
		// Write file type.
		if err := buf.WriteByte(entry.FileType); err != nil {
			return err
		}
		// Write file path length.
		pathBytes := []byte(entry.FilePath)
		pathLength := uint16(len(pathBytes))
		if err := binary.Write(buf, binary.BigEndian, pathLength); err != nil {
			return err
		}
		// Write file path.
		if _, err := buf.Write(pathBytes); err != nil {
			return err
		}
	}

	// Write the complete manifest buffer to the writer.
	_, err := w.Write(buf.Bytes())
	return err
}

func ReadManifest(r io.Reader) error {
	var magic uint32
	if err := binary.Read(r, binary.BigEndian, &magic); err != nil {
		return fmt.Errorf("error reading manifest magic: %v", err)
	}
	if magic != manifestMagicNumber {
		return fmt.Errorf("invalid manifest magic: expected 0x%X, got 0x%X", manifestMagicNumber, magic)
	}

	var version uint32
	if err := binary.Read(r, binary.BigEndian, &version); err != nil {
		return fmt.Errorf("error reading manifest version: %v", err)
	}
	if version != manifestVersion {
		return fmt.Errorf("unsupported manifest version: %d", version)
	}

	var entryCount uint64
	if err := binary.Read(r, binary.BigEndian, &entryCount); err != nil {
		return fmt.Errorf("error reading manifest entry count: %v", err)
	}

	fmt.Printf("Manifest contains %d entries:\n", entryCount)
	for i := uint64(0); i < entryCount; i++ {
		var headerOffset, fileSize uint64
		var fileType byte
		var pathLength uint16

		if err := binary.Read(r, binary.BigEndian, &headerOffset); err != nil {
			return fmt.Errorf("error reading header offset: %v", err)
		}
		if err := binary.Read(r, binary.BigEndian, &fileSize); err != nil {
			return fmt.Errorf("error reading file size: %v", err)
		}
		if err := binary.Read(r, binary.BigEndian, &fileType); err != nil {
			return fmt.Errorf("error reading file type: %v", err)
		}
		if err := binary.Read(r, binary.BigEndian, &pathLength); err != nil {
			return fmt.Errorf("error reading path length: %v", err)
		}

		pathBytes := make([]byte, pathLength)
		if _, err := io.ReadFull(r, pathBytes); err != nil {
			return fmt.Errorf("error reading path bytes: %v", err)
		}

		fmt.Printf("Entry %d:\n", i+1)
		fmt.Printf("  Header Offset: %d\n", headerOffset)
		fmt.Printf("  File Size: %d\n", fileSize)
		fmt.Printf("  File Type: %d\n", fileType)
		fmt.Printf("  File Path: %s\n", string(pathBytes))
	}

	// Read and validate manifest trailer.
	var trailer uint32
	if err := binary.Read(r, binary.BigEndian, &trailer); err != nil {
		return fmt.Errorf("error reading manifest trailer: %v", err)
	}
	if trailer != manifestMagicNumber {
		return fmt.Errorf("invalid manifest trailer: expected 0x%X, got 0x%X", manifestMagicNumber, trailer)
	}

	fmt.Println("Manifest read successfully.")
	return nil
}
