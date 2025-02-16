package dirstream

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
)

const (
	fileHeaderMagicNumber = 0x49525353
	headerVersion         = 1

	fileTypeRegular   = 0
	fileTypeDirectory = 1
	fileTypeSymlink   = 2
)

// fileHeader represents the header of a file in the stream.
type fileHeader struct {
	Version    uint32 // Header format version.
	FilePath   string // Relative file path.
	FileSize   uint64 // File size in bytes (0 for directories or symlinks).
	FileMode   uint32 // File mode.
	ModTime    int64  // Modification time (Unix timestamp).
	FileType   byte   // 0: regular file, 1: directory, 2: symlink.
	LinkTarget string // For symlinks, the target path.
}

// writeHeader builds a variable-length header, appends a 4-byte CRC, and writes it.
func writeHeader(w io.Writer, fh fileHeader) error {

	var buf bytes.Buffer

	// Write fixed fields: magic (4 bytes) and version (4 bytes).
	if err := binary.Write(&buf, binary.BigEndian, uint32(fileHeaderMagicNumber)); err != nil {
		return err
	}
	if err := binary.Write(&buf, binary.BigEndian, uint32(fh.Version)); err != nil {
		return err
	}

	// We'll write the header length (2 bytes) later.
	// Reserve 2 bytes for header length.
	if err := binary.Write(&buf, binary.BigEndian, uint16(0)); err != nil {
		return err
	}

	// Write file path.
	filePathBytes := []byte(fh.FilePath)
	filePathLen := uint16(len(filePathBytes))

	if filePathLen > 4096 {
		return fmt.Errorf("file path too long: %d", filePathLen)
	}

	if err := binary.Write(&buf, binary.BigEndian, filePathLen); err != nil {
		return err
	}
	if _, err := buf.Write(filePathBytes); err != nil {
		return err
	}

	if err := binary.Write(&buf, binary.BigEndian, fh.FileSize); err != nil {
		return err
	}
	if err := binary.Write(&buf, binary.BigEndian, fh.FileMode); err != nil {
		return err
	}
	if err := binary.Write(&buf, binary.BigEndian, uint64(fh.ModTime)); err != nil {
		return err
	}

	if err := buf.WriteByte(fh.FileType); err != nil {
		return err
	}

	if fh.FileType == fileTypeSymlink {
		linkTargetBytes := []byte(fh.LinkTarget)
		linkTargetLen := uint16(len(linkTargetBytes))

		if linkTargetLen > 4096 {
			return fmt.Errorf("Sys-link target too long: %d", linkTargetLen)
		}

		if err := binary.Write(&buf, binary.BigEndian, linkTargetLen); err != nil {
			return err
		}
		if _, err := buf.Write(linkTargetBytes); err != nil {
			return err
		}
	}

	// Header length can be written now and is defined as all bytes written so far (excluding the 4-byte CRC to be appended).
	headerLen := uint16(buf.Len())

	// Overwrite the header length field (which is at byte offset 8).
	// (Magic (4 bytes) + Version (4 bytes) = 8 bytes, so the header length field starts at offset 8.)
	binary.BigEndian.PutUint16(buf.Bytes()[8:10], headerLen)

	// Compute the CRC over the header bytes.
	crcValue := crc32.ChecksumIEEE(buf.Bytes())
	if err := binary.Write(&buf, binary.BigEndian, crcValue); err != nil {
		return err
	}

	// Write the complete header (variable-length header + 4-byte CRC) to the writer.
	_, err := w.Write(buf.Bytes())
	return err
}

// readHeader reads a variable-length header from the reader, verifies its CRC, and returns the parsed fileHeader.
func readHeader(r io.Reader) (fileHeader, error) {
	// read the fixed 10 bytes: magic (4), version (4), header length (2).
	fixed := make([]byte, 10)
	if _, err := io.ReadFull(r, fixed); err != nil {
		return fileHeader{}, fmt.Errorf("failed to read fixed header: %v", err)
	}

	magic := binary.BigEndian.Uint32(fixed[0:4])
	if magic != fileHeaderMagicNumber {
		return fileHeader{}, fmt.Errorf("invalid header magic: expected 0x%X, got 0x%X", fileHeaderMagicNumber, magic)
	}

	version := binary.BigEndian.Uint32(fixed[4:8])
	headerLen := binary.BigEndian.Uint16(fixed[8:10])
	if headerLen < 10 {
		return fileHeader{}, fmt.Errorf("invalid header length: %d", headerLen)
	}

	// Read the rest of the header (headerLen - 10 bytes).
	remainingHeader := make([]byte, int(headerLen)-10)
	if _, err := io.ReadFull(r, remainingHeader); err != nil {
		return fileHeader{}, fmt.Errorf("failed to read remaining header: %v", err)
	}

	// Combine fixed and remaining parts.
	fullHeader := append(fixed, remainingHeader...)

	// Read the 4-byte CRC that follows.
	crcBytes := make([]byte, 4)
	if _, err := io.ReadFull(r, crcBytes); err != nil {
		return fileHeader{}, fmt.Errorf("failed to read header CRC: %v", err)
	}
	storedCRC := binary.BigEndian.Uint32(crcBytes)
	computedCRC := crc32.ChecksumIEEE(fullHeader)
	if storedCRC != computedCRC {
		return fileHeader{}, fmt.Errorf("header CRC mismatch: expected 0x%X, got 0x%X", storedCRC, computedCRC)
	}

	var fh fileHeader
	fh.Version = version
	offset := 10

	// Read file path length (2 bytes).
	if offset+2 > len(fullHeader) {
		return fileHeader{}, fmt.Errorf("incomplete header (file path length)")
	}
	filePathLen := binary.BigEndian.Uint16(fullHeader[offset : offset+2])
	offset += 2

	// Read file path.
	if offset+int(filePathLen) > len(fullHeader) {
		return fileHeader{}, fmt.Errorf("incomplete header (file path)")
	}
	fh.FilePath = string(fullHeader[offset : offset+int(filePathLen)])
	offset += int(filePathLen)

	if offset+8 > len(fullHeader) {
		return fileHeader{}, fmt.Errorf("incomplete header (file size)")
	}
	fh.FileSize = binary.BigEndian.Uint64(fullHeader[offset : offset+8])
	offset += 8

	if offset+4 > len(fullHeader) {
		return fileHeader{}, fmt.Errorf("incomplete header (file mode)")
	}
	fh.FileMode = binary.BigEndian.Uint32(fullHeader[offset : offset+4])
	offset += 4

	if offset+8 > len(fullHeader) {
		return fileHeader{}, fmt.Errorf("incomplete header (modification time)")
	}
	fh.ModTime = int64(binary.BigEndian.Uint64(fullHeader[offset : offset+8]))
	offset += 8

	if offset+1 > len(fullHeader) {
		return fileHeader{}, fmt.Errorf("incomplete header (file type)")
	}
	fh.FileType = fullHeader[offset]
	offset += 1

	if fh.FileType == fileTypeSymlink {
		if offset+2 > len(fullHeader) {
			return fileHeader{}, fmt.Errorf("incomplete header (link target length)")
		}
		linkTargetLen := binary.BigEndian.Uint16(fullHeader[offset : offset+2])
		offset += 2
		if offset+int(linkTargetLen) > len(fullHeader) {
			return fileHeader{}, fmt.Errorf("incomplete header (link target)")
		}
		fh.LinkTarget = string(fullHeader[offset : offset+int(linkTargetLen)])
		offset += int(linkTargetLen)
	}

	return fh, nil
}
