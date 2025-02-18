package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/maja42/ember"
	"io"
	"lukasolson.net/common"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//go:embed run.bat
var runScript string

func bootstrap() error {

	exit := ValidateExecutableHash()
	if exit {
		return fmt.Errorf("error validating executable hash")
	}

	attachments, err := ember.Open()
	if err != nil {
		return err
	}
	defer attachments.Close()

	if ValidateHashes(attachments) {
		fmt.Println("Self-integrity validated successfully.")
	} else {
		return fmt.Errorf("error validating hashes")
	}

	settings, err := GetSettings(attachments)
	if err != nil {
		return err
	}

	// check if the bootstrap has already been run
	scriptExtractDir := *settings.ScriptExtractDir

	pythonExtractDir := *settings.PythonExtractDir

	integrityChecked := false

	if _, err := os.Stat("bootstrapped"); os.IsNotExist(err) {
		// if the bootstrap has not been run, extract the Python and program files

		fmt.Println("Performing first time setup...")

		fmt.Println("Reading embedded files...")

		PythonReader := attachments.Reader(common.PythonFilename)

		if PythonReader == nil {
			return fmt.Errorf("error reading Python. Ensure it is embedded in the binary")
		}

		PayloadReader := attachments.Reader(common.PayloadFilename)

		if PayloadReader == nil {
			return fmt.Errorf("error reading payload. Ensure it is embedded in the binary")
		}

		wheelsReader := attachments.Reader(common.WheelsFilename)
		if wheelsReader == nil {
			return fmt.Errorf("error reading wheels. Ensure it is embedded in the binary")
		}

		// extract the files to the disk
		fmt.Println("Extracting Python redistributable...")

		err = common.StreamToDir(PythonReader, pythonExtractDir)
		if err != nil {
			println("error extracting Python zip file")
			return err
		}

		fmt.Println("Extracting Scripts...")
		err = common.StreamToDir(PayloadReader, scriptExtractDir)
		if err != nil {
			println("error extracting payload zip file")
			return err
		}

		fmt.Println("Extracting Wheels...")

		wheelsDir := path.Join(pythonExtractDir, common.WheelsFilename)
		err = common.StreamToDir(wheelsReader, wheelsDir)
		if err != nil {
			println("error extracting wheels zip file")
			return err
		}

		fmt.Println("Extracted files successfully.")

		// Validate the integrity of the extracted files
		EmbeddedIntegrityHashes := attachments.Reader(common.IntegrityFilename)
		if EmbeddedIntegrityHashes == nil {
			panic("Error reading integrity hashes. Ensure they are embedded in the binary.")
		}

		integrityData, err := io.ReadAll(EmbeddedIntegrityHashes)
		if err != nil {
			panic("Error reading data from reader: " + err.Error())
		}

		err, isIntegral := VerifyExtractionIntegrity(integrityData, scriptExtractDir)

		if isIntegral != true {
			fmt.Println("Error validating integrity of extracted files. Please try again or contact the distributor.")

			// quit the program with an error code
			return fmt.Errorf("file integrity check failed")
		}

		integrityChecked = true

		// install the required packages

		fmt.Println("Installing required packages...")

		pythonPath := filepath.Join(pythonExtractDir, "python.exe")
		if err := common.RunCommand(pythonPath, []string{common.GetPipName(pythonExtractDir), "install", "pip", "setuptools", "wheel"}); err != nil {
			fmt.Println("Error installing packages")
			return err
		}

		// if requirements.txt exists, install the requirements
		requirementsFile := *settings.RequirementsFile
		if _, err := os.Stat(requirementsFile); err == nil {
			if err := common.RunCommand(pythonPath, []string{common.GetPipName(pythonExtractDir), "install", "--find-links",
				path.Join(wheelsDir) + "/", "--only-binary=:all:", "-r", requirementsFile}); err != nil {
				fmt.Println("Error while installing requirements from disk... ")
			}
		}

		// setup script path is relative to the extracted script directory
		setupScriptName := *settings.SetupScript

		// run the setup.py file if configured
		if setupScriptName != "" {
			setupScriptPath := path.Join(scriptExtractDir, setupScriptName)
			if err := common.RunCommand(pythonPath, []string{setupScriptPath}); err != nil {
				fmt.Println("Error running "+setupScriptName+":", err)
				return err
			}
		}

		myHash, err := calculateSelfHash()

		err = common.SaveContentsToFile("bootstrapped", myHash)
		if err != nil {
			fmt.Println("Error saving hash to file:", err)
		}

	}

	// Copy the files to the root directory if they are listed in the settings and they exist
	for _, file := range settings.FilesToCopyToRoot {
		filePath := path.Join(scriptExtractDir, file)
		if common.DoesPathExist(filePath) {
			err = common.CopyFile(filePath, file)
			if err != nil {
				fmt.Println("Error copying file to root")
				return err
			}
		}
	}

	if integrityChecked != true {
		// Validate the integrity of the extracted files
		EmbeddedIntegrityHashes := attachments.Reader(common.IntegrityFilename)
		if EmbeddedIntegrityHashes == nil {
			panic("Error reading integrity hashes. Ensure they are embedded in the binary.")
		}

		integrityData, err := io.ReadAll(EmbeddedIntegrityHashes)
		if err != nil {
			panic("Error reading data from reader: " + err.Error())
		}

		err, isIntegral := VerifyExtractionIntegrity(integrityData, scriptExtractDir)

		if *settings.RunAfterInstall == false && isIntegral != true {
			fmt.Println("Please rerun the installer to reinstall the environment.")
			os.Remove("bootstrapped")

			// quit the program with an error code
			return fmt.Errorf("file integrity check failed")
		}
	}

	// if main script is not set, exit, as there is nothing to run
	if *settings.MainScript == "" {
		fmt.Println("Files installed successfully. Exiting.")
		return nil
	} else {

		// Create the run.bat
		pythonExecutable := filepath.Join(pythonExtractDir, "python.exe")
		mainScriptPath := path.Join(scriptExtractDir, *settings.MainScript)

		// replace the placeholders in the runscript with the actual values
		runScript = strings.ReplaceAll(runScript, "{{PYTHON_EXE}}", pythonExecutable)
		runScript = strings.ReplaceAll(runScript, "{{MAIN_SCRIPT}}", mainScriptPath)
		runScript = strings.ReplaceAll(runScript, "{{SCRIPTS_DIR}}", scriptExtractDir)

		err = os.WriteFile("run.bat", []byte(runScript), 0644)

		runBatPath, err := filepath.Abs("run.bat")
		if err != nil {
			fmt.Println("Error getting absolute path for run.bat")
			return err
		}

		if *settings.RunAfterInstall {
			fmt.Println("Running script...")

			if err := common.RunCommand(runBatPath, os.Args[1:]); err != nil {
				fmt.Println("Error running script")
				return err
			}

			fmt.Println("Script completed.")
		} else {
			fmt.Println("Please run the following command in the command line to run the script:")
			fmt.Println(runBatPath)
		}

	}

	return nil
}

func VerifyExtractionIntegrity(integrityData []byte, scriptExtractDir string) (error, bool) {

	// these will be in the form of a json string, so we need to unmarshal them
	var fileHashes []common.FileHash

	// Unmarshal JSON string to slice of FileHash objects
	err := json.Unmarshal(integrityData, &fileHashes)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return nil, false
	}

	// get the hashes of the extracted files
	tamperedFiles, err := common.VerifyDirectoryIntegrity(scriptExtractDir, fileHashes)

	if err != nil {
		panic(err)
	}

	if len(tamperedFiles) > 0 {

		fmt.Println("Error validating integrity of extracted files.")
		fmt.Println("Warning, the following files have been modified since installation:")

		for _, file := range tamperedFiles {
			fmt.Println(" - " + file)
		}

		return nil, false

	} else {
		fmt.Println("Installation integrity validated successfully.")
	}
	return err, true
}

func ValidateExecutableHash() (exit bool) {
	myHash, err := calculateSelfHash()

	if err != nil {
		fmt.Println("Error calculating hash:", err)
		return true
	}

	if common.DoesPathExist("bootstrapped") {
		// read the hash from the file and compare it to the hash of the executable
		fileHash, err := os.ReadFile("bootstrapped")
		if err != nil {
			fmt.Println("Error reading hash file:", err)
			return true
		}

		if strings.TrimSpace(string(fileHash)) != myHash {
			fmt.Println("Error: Executable hash does not match previously accepted hash. File may have been tampered with.")

			fmt.Println("Expected:", string(fileHash))
			fmt.Println("Actual:", myHash)

			fmt.Println("Please validate the Md5 hash with the one supplied by the distributor before continuing")

			common.PressButtonToContinue("Press enter to accept the new hash and continue...")

			err = common.SaveContentsToFile("bootstrapped", myHash)
			if err != nil {
				fmt.Println("Error saving hash to file:", err)
				return true
			}

		} else {
			fmt.Println("Hashes match. File integrity validated.")
		}

	} else {

		fmt.Println("Please validate my Md5 hash before continuing")
		fmt.Println("While the hash is not a guarantee of safety, it is a good indicator of file integrity.")
		fmt.Println("You can validate my hash by running the following command in the command line:")

		var exeName string

		// check if os.Args[0] has .exe extension
		if !strings.HasSuffix(os.Args[0], ".exe") {
			exeName = os.Args[0] + ".exe"
		} else {
			exeName = os.Args[0]
		}

		fmt.Println("certutil -hashfile", exeName, "MD5")
		fmt.Println("Note: If hash values do not match, the file may have been tampered with.")

		common.PressButtonToContinue("Press enter to continue...")
	}
	return false
}

func calculateSelfHash() (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path:", err)
		return "", err
	}

	myHash, err := common.Md5SumFile(executablePath)

	if err != nil {
		fmt.Println("Error getting hash of executable:", err)
		return "", err
	}
	return myHash, err
}

func GetSettings(attachments *ember.Attachments) (common.PythonSetupSettings, error) {
	ConfigReader := attachments.Reader(common.GetConfigEmbedName())

	if ConfigReader == nil {
		fmt.Println("Error reading config. Ensure it is embedded in the binary.")
		return common.PythonSetupSettings{}, fmt.Errorf("error reading config. Ensure it is embedded in the binary")
	}
	config, err := io.ReadAll(ConfigReader)

	var settings common.PythonSetupSettings
	err = json.Unmarshal(config, &settings)
	return settings, err
}

func GetHashmap(attachments *ember.Attachments) (map[string]string, error) {
	HashReader := attachments.Reader(common.HashesFilename)
	if HashReader == nil {
		fmt.Println("Error reading hash. Ensure it is embedded in the binary.")

		// throw a new error to prevent further execution
		return nil, fmt.Errorf("error reading hash. Ensure it is embedded in the binary")
	}

	hash, err := io.ReadAll(HashReader)

	if err != nil {
		fmt.Println("Error reading hash:", err)
		return nil, err
	}

	var hashMap map[string]string

	err = json.Unmarshal(hash, &hashMap)

	if err != nil {
		fmt.Println("Error unmarshalling hash:", err)
		return nil, err
	}

	return hashMap, nil
}

func ValidateHash(seeker io.ReadSeeker, expectedHash string) (actualHash string, equal bool) {
	actualHash, err := common.HashReadSeeker(seeker)
	if err != nil {
		fmt.Println("Error reading hash:", err)
		return "", false
	}

	if actualHash != expectedHash {
		return actualHash, false
	}

	return actualHash, true
}

func ValidateHashes(attachments *ember.Attachments) bool {

	attachmentList := attachments.List()

	hashMap, err := GetHashmap(attachments)
	if err != nil {
		return false
	}

	allHashesMatch := true

	for _, attachment := range attachmentList {
		if attachment == common.HashesFilename {
			continue
		}

		attachmentReader := attachments.Reader(attachment)

		if attachmentReader == nil {
			fmt.Println("Error reading attachment:", attachment)
			return false
		}

		actualHash, hashesMatch := ValidateHash(attachmentReader, hashMap[attachment])

		if !hashesMatch {
			fmt.Println("Error validating hash for:", attachment, " -> Expected:", hashMap[attachment], "Actual:", actualHash)
			allHashesMatch = false
		}
	}

	return allHashesMatch
}
