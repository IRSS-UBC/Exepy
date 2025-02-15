package dirstream

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// Updated header size to 512 bytes.
	headerSize = 512

	// Default chunk size if not specified.
	DefaultChunkSize = 4096

	// Each chunk is preceded by a header:
	// 4 bytes for the magic number and 8 bytes for the chunk length.
	chunkHeaderSize = 4 + 8

	// Magic number for chunk header identification.
	magicNumber = 0xDEADBEEF

	// Header version for our file header format.
	headerVersion = 1
)

const (
	FileTypeRegular   = 0
	FileTypeDirectory = 1
	FileTypeSymlink   = 2
)

// FileHeader represents the header of a file in the stream.
type fileHeader struct {
	Version    uint32 // Header format version.
	FilePath   string // Relative file path (max 256 bytes including null terminator).
	FileSize   uint64 // File size in bytes (0 for directories or symlinks).
	FileMode   uint32 // File mode.
	ModTime    int64  // Modification time (Unix timestamp).
	FileType   byte   // 0: regular file, 1: directory, 2: symlink.
	LinkTarget string // For symlinks, the target path (max 128 bytes including null terminator).
}

// -----------------------------------------------------------------------------
// Header Serialization / Deserialization
// -----------------------------------------------------------------------------

// writeHeader writes a fixed 512-byte header to the writer.
func writeHeader(w io.Writer, fh fileHeader) error {
	headerBytes := make([]byte, headerSize)

	// Bytes 0-3: Header version.
	binary.BigEndian.PutUint32(headerBytes[0:4], fh.Version)

	// Bytes 4-259: File path (null-terminated).
	pathBytes := []byte(fh.FilePath)
	if len(pathBytes) >= 256 {
		return fmt.Errorf("file path too long: %s", fh.FilePath)
	}
	copy(headerBytes[4:4+len(pathBytes)], pathBytes)
	headerBytes[4+len(pathBytes)] = 0 // Null terminator.

	// Bytes 260-267: File size.
	binary.BigEndian.PutUint64(headerBytes[260:268], fh.FileSize)

	// Bytes 268-271: File mode.
	binary.BigEndian.PutUint32(headerBytes[268:272], fh.FileMode)

	// Bytes 272-279: Modification time.
	binary.BigEndian.PutUint64(headerBytes[272:280], uint64(fh.ModTime))

	// Byte 280: File type.
	headerBytes[280] = fh.FileType

	// Bytes 281-408: Symlink target (if applicable; null-terminated).
	if fh.FileType == FileTypeSymlink {
		targetBytes := []byte(fh.LinkTarget)
		if len(targetBytes) >= 128 {
			return fmt.Errorf("symlink target too long: %s", fh.LinkTarget)
		}
		copy(headerBytes[281:281+len(targetBytes)], targetBytes)
		headerBytes[281+len(targetBytes)] = 0 // Null terminator.
	}

	// The remaining bytes are reserved (zeroed by default).
	_, err := w.Write(headerBytes)
	return err
}

// readHeader reads and parses a 512-byte file header from the reader.
func readHeader(r io.Reader) (fileHeader, error) {
	headerBytes := make([]byte, headerSize)
	if _, err := io.ReadFull(r, headerBytes); err != nil {
		return fileHeader{}, err
	}

	fh := fileHeader{}
	fh.Version = binary.BigEndian.Uint32(headerBytes[0:4])

	// Bytes 4-259: File path.
	pathData := headerBytes[4:260]
	zeroIndex := bytes.IndexByte(pathData, 0)
	if zeroIndex == -1 {
		zeroIndex = len(pathData)
	}
	fh.FilePath = string(pathData[:zeroIndex])

	// Bytes 260-267: File size.
	fh.FileSize = binary.BigEndian.Uint64(headerBytes[260:268])

	// Bytes 268-271: File mode.
	fh.FileMode = binary.BigEndian.Uint32(headerBytes[268:272])

	// Bytes 272-279: Modification time.
	fh.ModTime = int64(binary.BigEndian.Uint64(headerBytes[272:280]))

	// Byte 280: File type.
	fh.FileType = headerBytes[280]

	// Bytes 281-408: Symlink target (if applicable).
	if fh.FileType == FileTypeSymlink {
		targetData := headerBytes[281:409]
		zeroIndex = bytes.IndexByte(targetData, 0)
		if zeroIndex == -1 {
			zeroIndex = len(targetData)
		}
		fh.LinkTarget = string(targetData[:zeroIndex])
	}

	return fh, nil
}
