package main

import (
	"dirstream"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// main provides a simple interactive interface for encoding and decoding.
func main() {
	interactiveTest()
}

// interactiveTest provides an interactive way to test encoding/decoding.
func interactiveTest() {
	fmt.Println("\nInteractive Test:")
	fmt.Println("1. Encode a directory")
	fmt.Println("2. Decode a stream")
	fmt.Println("3. Exit")

	var choice string
	fmt.Print("Enter your choice: ")
	fmt.Scanln(&choice)

	switch choice {
	case "1":
		var sourceDir string
		fmt.Print("Enter source directory: ")
		fmt.Scanln(&sourceDir)
		// Sanitize input.
		sourceDir = filepath.Clean(sourceDir)
		if !strings.HasPrefix(sourceDir, "/") {
			currentDir, _ := os.Getwd()
			sourceDir = filepath.Join(currentDir, sourceDir)
		}
		if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Source directory does not exist: %s\n", sourceDir)
			return
		}

		encoder := dirstream.NewEncoder(sourceDir, dirstream.DefaultChunkSize)
		stream, err := encoder.Encode()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Encoding error: %v\n", err)
			return
		}

		var outputFile string
		fmt.Print("Enter output file name (or leave blank for output.stream): ")
		fmt.Scanln(&outputFile)
		if outputFile == "" {
			outputFile = "output.stream"
		}
		outputFile = filepath.Clean(outputFile)
		if !strings.HasPrefix(outputFile, "/") {
			currentDir, _ := os.Getwd()
			outputFile = filepath.Join(currentDir, outputFile)
		}

		f, err := os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			return
		}
		defer f.Close()

		_, err = io.Copy(f, stream)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to output file: %v\n", err)
			return
		}
		fmt.Println("Encoding successful. Output written to", outputFile)

	case "2":
		var inputFile string
		fmt.Print("Enter input file name (or leave blank for output.stream): ")
		fmt.Scanln(&inputFile)
		if inputFile == "" {
			inputFile = "output.stream"
		}
		inputFile = filepath.Clean(inputFile)
		if !strings.HasPrefix(inputFile, "/") {
			currentDir, _ := os.Getwd()
			inputFile = filepath.Join(currentDir, inputFile)
		}

		f, err := os.Open(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
			return
		}
		defer f.Close()

		var destDir string
		fmt.Print("Enter destination directory: ")
		fmt.Scanln(&destDir)
		destDir = filepath.Clean(destDir)
		if !strings.HasPrefix(destDir, "/") {
			currentDir, _ := os.Getwd()
			destDir = filepath.Join(currentDir, destDir)
		}

		// Create decoder with strict mode disabled.
		decoder := dirstream.NewDecoder(destDir, false, dirstream.DefaultChunkSize)
		err = decoder.Decode(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Decoding error: %v\n", err)
			return
		}
		fmt.Println("Decoding successful. Files written to", destDir)

	case "3":
		return

	default:
		fmt.Println("Invalid choice.")
	}
}
