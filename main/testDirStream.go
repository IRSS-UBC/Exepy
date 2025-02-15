package main

import (
	"dirstream"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// main provides a command-line interface for encoding and decoding.
func main() {
	// Define command-line flags.
	encodeCmd := flag.NewFlagSet("encode", flag.ExitOnError)
	encodeSourceDir := encodeCmd.String("source", "", "Source directory to encode (required)")
	encodeOutputFile := encodeCmd.String("output", "output.stream", "Output file name")

	decodeCmd := flag.NewFlagSet("decode", flag.ExitOnError)
	decodeInputFile := decodeCmd.String("input", "output.stream", "Input file to decode")
	decodeDestDir := decodeCmd.String("dest", "", "Destination directory (required)")
	decodeStrictMode := decodeCmd.Bool("strict", false, "Enable strict mode for decoding")

	if len(os.Args) < 2 {
		fmt.Println("Usage: dirstream <command> [options]")
		fmt.Println("Commands: encode, decode")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "encode":
		encodeCmd.Parse(os.Args[2:])
		if *encodeSourceDir == "" {
			fmt.Fprintln(os.Stderr, "Error: Source directory is required for encoding.")
			encodeCmd.PrintDefaults()
			os.Exit(1)
		}
		encode(*encodeSourceDir, *encodeOutputFile)

	case "decode":
		decodeCmd.Parse(os.Args[2:])
		if *decodeDestDir == "" {
			fmt.Fprintln(os.Stderr, "Error: Destination directory is required for decoding.")
			decodeCmd.PrintDefaults()
			os.Exit(1)
		}
		decode(*decodeInputFile, *decodeDestDir, *decodeStrictMode)

	default:
		fmt.Println("Usage: dirstream <command> [options]")
		fmt.Println("Commands: encode, decode")
		os.Exit(1)
	}
}

// encode handles the encoding process.
func encode(sourceDir, outputFile string) {
	// Sanitize input
	sourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid source directory: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Source directory does not exist: %s\n", sourceDir)
		os.Exit(1)
	}

	encoder := dirstream.NewEncoder(sourceDir, dirstream.DefaultChunkSize)
	stream, err := encoder.Encode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Encoding error: %v\n", err)
		os.Exit(1)
	}

	outputFile, err = filepath.Abs(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid output file path: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	_, err = io.Copy(f, stream)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to output file: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Encoding successful. Output written to", outputFile)
}

// decode handles the decoding process.
func decode(inputFile, destDir string, strictMode bool) {
	inputFile, err := filepath.Abs(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid input file path: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Open(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	destDir, err = filepath.Abs(destDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid destination directory: %v\n", err)
		os.Exit(1)
	}

	decoder := dirstream.NewDecoder(destDir, strictMode, dirstream.DefaultChunkSize) // Use strictMode from arguments
	err = decoder.Decode(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Decoding error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Decoding successful. Files written to", destDir)
}
