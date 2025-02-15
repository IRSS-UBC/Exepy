package dirstream

import (
	"bytes"
	"encoding/binary"
	"io"
)

const (
	manifestMagic   = 0xABCD1234
	manifestVersion = 1
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

	// Write manifest header.
	if err := binary.Write(buf, binary.BigEndian, uint32(manifestMagic)); err != nil {
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

	// Optionally, write a manifest trailer (e.g., same magic number) to signal the end.
	if err := binary.Write(buf, binary.BigEndian, uint32(manifestMagic)); err != nil {
		return err
	}

	// Write the complete manifest buffer to the writer.
	_, err := w.Write(buf.Bytes())
	return err
}
