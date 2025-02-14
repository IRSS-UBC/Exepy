package windowsPE

import (
	"encoding/binary"
	"errors"
)

// removeSignature zeros out the security directory and checksum in a PE file.
func RemoveSignature(peBytes []byte) ([]byte, error) {
	// A valid DOS header is at least 64 bytes.
	if len(peBytes) < 64 {
		return nil, errors.New("file is too small to be a PE file")
	}

	// 1. Parse the DOS Header to get the offset to the PE header.
	peOffset := int(binary.LittleEndian.Uint32(peBytes[0x3C : 0x3C+4]))

	// Ensure the PE header is within bounds.
	if len(peBytes) < peOffset+4 {
		return nil, errors.New("file is too small to be a PE file")
	}

	// 2. Verify the PE signature ("PE\0\0").
	if string(peBytes[peOffset:peOffset+4]) != "PE\x00\x00" {
		return nil, errors.New("invalid PE signature")
	}

	// Calculate offsets:
	fileHeaderOffset := peOffset + 4
	optionalHeaderOffset := fileHeaderOffset + 20

	// Make sure we have at least the magic number from the optional header.
	if len(peBytes) < optionalHeaderOffset+2 {
		return nil, errors.New("file does not have an optional header")
	}

	// Read the magic number to determine PE format.
	magic := binary.LittleEndian.Uint16(peBytes[optionalHeaderOffset : optionalHeaderOffset+2])
	var dataDirectoryOffset int
	var optionalHeaderCheckSumOffset int

	switch magic {
	case 0x10b: // PE32
		// For PE32, the data directories start at offset 96.
		if len(peBytes) < optionalHeaderOffset+96 {
			return nil, errors.New("optional header too small for PE32")
		}
		dataDirectoryOffset = optionalHeaderOffset + 96
		optionalHeaderCheckSumOffset = optionalHeaderOffset + 64
	case 0x20b: // PE32+
		// For PE32+, the data directories start at offset 112.
		if len(peBytes) < optionalHeaderOffset+112 {
			return nil, errors.New("optional header too small for PE32+")
		}
		dataDirectoryOffset = optionalHeaderOffset + 112
		optionalHeaderCheckSumOffset = optionalHeaderOffset + 64
	default:
		return nil, errors.New("unknown optional header magic")
	}

	const ImageDirectoryEntrySecurity = 4
	// Each data directory entry is 8 bytes (4 bytes VirtualAddress, 4 bytes Size).
	securityDirectoryOffset := dataDirectoryOffset + (ImageDirectoryEntrySecurity * 8)

	// Check bounds before modifying the file.
	if securityDirectoryOffset+8 > len(peBytes) {
		return nil, errors.New("security directory offset out of bounds")
	}
	if optionalHeaderCheckSumOffset+4 > len(peBytes) {
		return nil, errors.New("optional header checksum offset out of bounds")
	}

	// 3. Zero out the Security Directory (Digital Signature).
	binary.LittleEndian.PutUint32(peBytes[securityDirectoryOffset:securityDirectoryOffset+4], 0)   // VirtualAddress
	binary.LittleEndian.PutUint32(peBytes[securityDirectoryOffset+4:securityDirectoryOffset+8], 0) // Size

	// 4. Zero out the Checksum Value.
	binary.LittleEndian.PutUint32(peBytes[optionalHeaderCheckSumOffset:optionalHeaderCheckSumOffset+4], 0)

	return peBytes, nil
}
