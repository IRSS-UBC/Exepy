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

	HeaderMagicNumber = 0x49525353 // 4-byte magic string

	// Each chunk is preceded by a header:
	// 4 bytes for the magic number and 8 bytes for the chunk length.
	chunkHeaderSize = 4 + 8

	// Magic number for chunk header identification.
	chunkMagicNumber = 0xDEADBEEF

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

func writeHeader(w io.Writer, fh fileHeader) error {
	headerBytes := make([]byte, headerSize)

	// Bytes 0-3: Magic string.
	binary.BigEndian.PutUint32(headerBytes[0:4], HeaderMagicNumber)

	// Bytes 4-7: Header version.
	binary.BigEndian.PutUint32(headerBytes[4:8], fh.Version)

	// Bytes 8-263: File path (null-terminated).
	pathBytes := []byte(fh.FilePath)
	if len(pathBytes) >= 256 {
		return fmt.Errorf("file path too long: %s", fh.FilePath)
	}
	copy(headerBytes[8:8+len(pathBytes)], pathBytes)
	headerBytes[8+len(pathBytes)] = 0 // Null terminator.

	// Continue writing file size, file mode, mod time, file type, etc.
	// For example:
	binary.BigEndian.PutUint64(headerBytes[260:268], fh.FileSize)
	binary.BigEndian.PutUint32(headerBytes[268:272], fh.FileMode)
	binary.BigEndian.PutUint64(headerBytes[272:280], uint64(fh.ModTime))
	headerBytes[280] = fh.FileType

	// Symlink target and reserved bytes as before.
	if fh.FileType == FileTypeSymlink {
		targetBytes := []byte(fh.LinkTarget)
		if len(targetBytes) >= 128 {
			return fmt.Errorf("symlink target too long: %s", fh.LinkTarget)
		}
		copy(headerBytes[281:281+len(targetBytes)], targetBytes)
		headerBytes[281+len(targetBytes)] = 0 // Null terminator.
	}

	_, err := w.Write(headerBytes)
	return err
}

func readHeader(r io.Reader) (fileHeader, error) {
	headerBytes := make([]byte, headerSize)
	if _, err := io.ReadFull(r, headerBytes); err != nil {
		return fileHeader{}, err
	}

	// Validate magic string.
	if binary.BigEndian.Uint32(headerBytes[0:4]) != HeaderMagicNumber {
		return fileHeader{}, fmt.Errorf("invalid header magic: expected %d, got %d", HeaderMagicNumber, binary.BigEndian.Uint32(headerBytes[0:4]))
	}

	fh := fileHeader{}
	fh.Version = binary.BigEndian.Uint32(headerBytes[4:8])

	// Read file path from bytes 8-263.
	pathData := headerBytes[8:264]
	zeroIndex := bytes.IndexByte(pathData, 0)
	if zeroIndex == -1 {
		zeroIndex = len(pathData)
	}
	fh.FilePath = string(pathData[:zeroIndex])

	// Read remaining fields as before.
	fh.FileSize = binary.BigEndian.Uint64(headerBytes[260:268])
	fh.FileMode = binary.BigEndian.Uint32(headerBytes[268:272])
	fh.ModTime = int64(binary.BigEndian.Uint64(headerBytes[272:280]))
	fh.FileType = headerBytes[280]

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
