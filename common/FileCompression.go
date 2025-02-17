package common

import (
	"bytes"
	"compress/gzip"
	"dirstream"
	"fmt"
	"io"
)

func DirToStream(directoryPath string, ignoredDirs []string) (io.ReadSeeker, error) {
	files, err := dirstream.BuildRelativeFileList(directoryPath, ignoredDirs)
	if err != nil {
		return nil, fmt.Errorf("failed to build file list: %w", err)
	}

	encoder := dirstream.NewEncoder(directoryPath, dirstream.DefaultChunkSize)
	encoderStream, err := encoder.Encode(files)
	if err != nil {
		return nil, fmt.Errorf("failed to encode directory: %w", err)
	}

	var buf bytes.Buffer

	gzipWriter := gzip.NewWriter(&buf)

	if _, err := io.Copy(gzipWriter, encoderStream); err != nil {
		gzipWriter.Close() // ensure we close on error
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}

	// Close the gzip writer to flush all data into the buffer.
	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return bytes.NewReader(buf.Bytes()), nil
}

func StreamToDir(IOReader io.Reader, outputDir string) error {
	// Create a gzip reader from the input stream.
	gzipReader, err := gzip.NewReader(IOReader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Pass the decompressed stream to the dirstream decoder.
	if err := dirstream.NewDecoder(outputDir, false, dirstream.DefaultChunkSize).Decode(gzipReader); err != nil {
		return fmt.Errorf("failed to decode stream: %w", err)
	}

	return nil
}
