package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/maja42/ember/embedding"
	"io"
	"lukasolson.net/common"
	"os"
	"path"
)

const settingsFileName = "settings.json"

func createInstaller() {

	settings, err := common.LoadOrSaveDefault(settingsFileName)
	if err != nil {
		return
	}

	pythonScriptPath := path.Join(settings.ScriptDir, settings.MainScript)
	requirementsPath := path.Join(settings.ScriptDir, settings.RequirementsFile)

	// check if payload directory exists
	if !common.DoesPathExist(settings.ScriptDir) {
		println("Scripts directory does not exist: ", settings.ScriptDir)
		return
	}

	// check if payload directory has the main file
	if !common.DoesPathExist(pythonScriptPath) {
		println("Main file does not exist: ", pythonScriptPath)
		return
	}

	// if requirements file is listed, check that it exists
	if settings.RequirementsFile != "" {
		if !common.DoesPathExist(requirementsPath) {
			println("Requirements file is listed in config but does not exist: ", requirementsPath)
			return
		}
	}

	file, err := os.Create("bootstrap.exe")
	if err != nil {
		panic(err)
	}

	defer file.Close()

	pythonFile, wheelsFile, err := PreparePython(*settings)
	if err != nil {
		panic(err)
	}

	PayloadFile, err := common.CompressDirToStream(settings.ScriptDir)
	if err != nil {
		panic(err)
	}

	SettingsFile, err := os.Open(settingsFileName)
	defer SettingsFile.Close()

	embedMap := createEmbedMap(pythonFile, PayloadFile, wheelsFile, SettingsFile)

	if err := writePythonExecutable(file, embedMap); err != nil {
		return
	}

	file.Close()

	outputExeHash, err := common.Md5SumFile(file.Name())

	if err != nil {
		panic(err)
	}

	println("Output executable hash: ", outputExeHash, " saved to hash.txt")

	// save the hash to a file

	if err := common.SaveContentsToFile("hash.txt", outputExeHash); err != nil {
		println("Error saving hash to file")
	}

	println("Embedded payload")

}

func createEmbedMap(PythonRS, PayloadRS, wheelsFile, SettingsFile io.ReadSeeker) map[string]io.ReadSeeker {

	hashMap, hashBytes := HashFiles(PythonRS, PayloadRS, wheelsFile, SettingsFile)

	json.NewEncoder(hashBytes).Encode(hashMap)

	embedMap := make(map[string]io.ReadSeeker)

	embedMap[common.HashesEmbedName] = bytes.NewReader(hashBytes.Bytes())
	embedMap[common.PythonFilename] = PythonRS
	embedMap[common.PayloadFilename] = PayloadRS
	embedMap[common.WheelsFilename] = wheelsFile
	embedMap[common.GetConfigEmbedName()] = SettingsFile

	return embedMap
}

func HashFiles(PythonRS io.ReadSeeker, PayloadRS io.ReadSeeker, wheelsFile io.ReadSeeker, SettingsFile io.ReadSeeker) (map[string]string, *bytes.Buffer) {
	PythonHash, err := common.HashReadSeeker(PythonRS)
	if err != nil {
		panic(err)
	}

	PayloadHash, err := common.HashReadSeeker(PayloadRS)
	if err != nil {
		panic(err)
	}

	wheelsFileHash, err := common.HashReadSeeker(wheelsFile)
	if err != nil {
		panic(err)
	}

	SettingsFileHash, err := common.HashReadSeeker(SettingsFile)
	if err != nil {
		panic(err)
	}

	hashMap, hashBytes := make(map[string]string), new(bytes.Buffer)
	hashMap[common.PythonFilename] = PythonHash
	hashMap[common.PayloadFilename] = PayloadHash
	hashMap[common.WheelsFilename] = wheelsFileHash
	hashMap[common.GetConfigEmbedName()] = SettingsFileHash

	// print the hashes
	for k, v := range hashMap {
		fmt.Println("Hash for", k, ":", v)
	}

	return hashMap, hashBytes
}

// writePythonExecutable is a function that embeds attachments into a Python executable.
// It takes two parameters:
// - writer: an io.Writer where the resulting executable will be written.
// - attachments: a map where the key is the name of the attachment and the value is an io.ReadSeeker that reads the attachment's content.
func writePythonExecutable(writer io.Writer, attachments map[string]io.ReadSeeker) error {
	// Load the executable file of the current running program
	executableBytes, err := loadSelf()
	// If an error occurred while loading the executable, return
	if err != nil {
		return err
	}

	// Clean the executable file from any previous attachments
	exeWithoutSignature, err := removeSignature(executableBytes)

	if err != nil {
		return err
	}

	exeWithoutEmbeddings, err := removeEmbedding(exeWithoutSignature)

	if err != nil {
		return err
	}

	// Create a new reader for the executable bytes
	reader := bytes.NewReader(exeWithoutEmbeddings)

	// Embed the attachments into the executable
	err = embedding.Embed(writer, reader, attachments, nil)
	// If an error occurred while embedding the attachments, return
	if err != nil {
		return err
	}

	return nil
}

// loadSelf is a function that retrieves the executable file of the current running program.
// It returns the file content as a byte slice and an error if any occurred during the process.
func loadSelf() ([]byte, error) {
	// Get the path of the executable file
	selfPath, err := os.Executable()
	// If an error occurred while getting the path, return the error
	if err != nil {
		return nil, err
	}

	// Open the executable file
	file, err := os.Open(selfPath)
	// If an error occurred while opening the file, return the error
	if err != nil {
		return nil, err
	}
	// Ensure the file will be closed at the end of the function
	defer file.Close()

	// Create a new buffer to hold the file content
	memSlice := new(bytes.Buffer)

	// Copy the file content into the buffer
	_, err = io.Copy(memSlice, file)
	// If an error occurred while copying the file content, return the error
	if err != nil {
		return nil, err
	}

	// Return the file content as a byte slice and any error that might have occurred
	return memSlice.Bytes(), err
}

// removeSignature zeros out the security directory and checksum in a PE file.
func removeSignature(peBytes []byte) ([]byte, error) {
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

func removeEmbedding(file []byte) ([]byte, error) {

	out := new(bytes.Buffer)

	reader := bytes.NewReader(file)

	err := embedding.RemoveEmbedding(out, reader, nil)

	if errors.Is(err, embedding.ErrNothingEmbedded) {
		return file, nil
	}

	return out.Bytes(), err
}
