package dirstream

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"unsafe"
)

const (
	fileHeaderMagicNumber = 0x49525353
	headerSize            = 512
	headerVersion         = 1
)

const (
	fileTypeRegular   = 0
	fileTypeDirectory = 1
	fileTypeSymlink   = 2
)

// fileHeader represents the header of a file in the stream.
type fileHeader struct {
	Version    uint32
	FilePath   string
	FileSize   uint64
	FileMode   uint32
	ModTime    int64
	FileType   byte
	LinkTarget string
}

// writeHeader writes a file header to the writer and appends a CRC computed over the header (excluding the CRC field).
func writeHeader(w io.Writer, fh fileHeader) error {
	headerBytes := make([]byte, headerSize)

	// Bytes 0-3: Magic number.
	binary.BigEndian.PutUint32(headerBytes[0:0+4], fileHeaderMagicNumber)

	// Bytes 4-7: Header version.
	binary.BigEndian.PutUint32(headerBytes[4:4+4], fh.Version)

	// Bytes 8-263: File path (null-terminated).
	pathBytes := []byte(fh.FilePath)
	if len(pathBytes) >= 256 {
		return fmt.Errorf("file path too long: %s", fh.FilePath)
	}
	copy(headerBytes[8:8+len(pathBytes)], pathBytes)
	headerBytes[8+len(pathBytes)] = 0 // Null terminator.

	// Bytes 264-271: File size.
	binary.BigEndian.PutUint64(headerBytes[(8+256):8+256+8], fh.FileSize)

	// Bytes 272-275: File mode.
	binary.BigEndian.PutUint32(headerBytes[(8+256+8):8+256+8+4], fh.FileMode)

	// Bytes 276-283: Modification time.
	binary.BigEndian.PutUint64(headerBytes[(8+256+8+4):8+256+8+4+8], *(*uint64)(unsafe.Pointer(&fh.ModTime)))

	// Byte 284: File type.
	headerBytes[(8 + 256 + 8 + 4 + 8)] = fh.FileType

	// Bytes 285-412: Link target for symlinks.
	if fh.FileType == fileTypeSymlink {
		targetBytes := []byte(fh.LinkTarget)
		if len(targetBytes) >= 128 {
			return fmt.Errorf("symlink target too long: %s", fh.LinkTarget)
		}
		copy(headerBytes[(8+256+8+4+8+1):8+256+8+4+8+1+len(targetBytes)], targetBytes)
		headerBytes[8+256+8+4+8+1+len(targetBytes)] = 0 // Null terminator.
	}

	// Reserved area (bytes 413-507) is left as zero.
	// Compute CRC32 over header bytes from 0 to 507 (all except the last 4 bytes reserved for CRC).
	crcValue := crc32.ChecksumIEEE(headerBytes[:(headerSize - 4)])
	binary.BigEndian.PutUint32(headerBytes[(headerSize-4):headerSize-4+4], crcValue)

	_, err := w.Write(headerBytes)
	return err
}

// readHeader reads the header from the reader, verifies its CRC, and returns the parsed fileHeader.
func readHeader(r io.Reader) (fileHeader, error) {
	headerBytes := make([]byte, headerSize)
	if _, err := io.ReadFull(r, headerBytes); err != nil {
		return fileHeader{}, err
	}

	// Verify the CRC32 checksum.
	storedCRC := binary.BigEndian.Uint32(headerBytes[(headerSize - 4) : headerSize-4+4])
	calculatedCRC := crc32.ChecksumIEEE(headerBytes[:(headerSize - 4)])
	if storedCRC != calculatedCRC {
		return fileHeader{}, fmt.Errorf("header CRC mismatch: expected %x, got %x", storedCRC, calculatedCRC)
	}

	// Validate magic number.
	if binary.BigEndian.Uint32(headerBytes[0:0+4]) != fileHeaderMagicNumber {
		return fileHeader{}, fmt.Errorf("invalid header magic number: expected %d, got %d", fileHeaderMagicNumber, binary.BigEndian.Uint32(headerBytes[0:0+4]))
	}

	var fh fileHeader
	fh.Version = binary.BigEndian.Uint32(headerBytes[4 : 4+4])

	// Read file path from bytes 8-263.
	pathData := headerBytes[8 : 8+256]
	if zeroIndex := bytes.IndexByte(pathData, 0); zeroIndex != -1 {
		fh.FilePath = string(pathData[:zeroIndex])
	} else {
		fh.FilePath = string(pathData)
	}

	// Read file size.
	fh.FileSize = binary.BigEndian.Uint64(headerBytes[(8 + 256) : 8+256+8])

	// Read file mode.
	fh.FileMode = binary.BigEndian.Uint32(headerBytes[(8 + 256 + 8) : 8+256+8+4])

	// Read modification time.
	uModTime := binary.BigEndian.Uint64(headerBytes[(8 + 256 + 8 + 4) : 8+256+8+4+8])
	fh.ModTime = *(*int64)(unsafe.Pointer(&uModTime))

	// Read file type.
	fh.FileType = headerBytes[(8 + 256 + 8 + 4 + 8)]

	// If symlink, read link target.
	if fh.FileType == fileTypeSymlink {
		targetData := headerBytes[(8 + 256 + 8 + 4 + 8 + 1) : 8+256+8+4+8+1+128]
		if zeroIndex := bytes.IndexByte(targetData, 0); zeroIndex != -1 {
			fh.LinkTarget = string(targetData[:zeroIndex])
		} else {
			fh.LinkTarget = string(targetData)
		}
	}

	return fh, nil
}
