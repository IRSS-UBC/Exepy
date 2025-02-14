package main

import (
	"bufio"
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
	"time"
)

//go:embed run.bat
var runScript string

func bootstrap(pure bool) {

	exit := ValidateExecutableHash()
	if exit {
		return
	}

	attachments, err := ember.Open()
	if err != nil {
		fmt.Println("Error opening attachments:", err)
		return
	}
	defer attachments.Close()

	if ValidateHashes(attachments) {
		fmt.Println("Self-integrity validated successfully.")
	} else {
		fmt.Println("Error validating hashes.")
		return
	}

	settings, err := GetSettings(attachments)
	if err != nil {
		fmt.Println("Error reading settings:", err)
		return
	}

	// check if the bootstrap has already been run
	if _, err := os.Stat("bootstrapped"); os.IsNotExist(err) {
		// if the bootstrap has not been run, extract the Python and program files

		fmt.Println("Performing first time setup...")

		PythonReader := attachments.Reader(common.PythonFilename)

		if PythonReader == nil {
			fmt.Println("Error reading Python. Ensure it is embedded in the binary.")
			return
		}

		PayloadReader := attachments.Reader(common.PayloadFilename)

		if PayloadReader == nil {
			fmt.Println("Error reading payload. Ensure it is embedded in the binary.")
			return
		}

		wheelsReader := attachments.Reader(common.WheelsFilename)
		if wheelsReader == nil {
			fmt.Println("Error reading wheels. Ensure it is embedded in the binary.")
			return
		}

		err = common.DecompressIOStream(PythonReader, settings.PythonExtractDir)
		if err != nil {
			fmt.Println("Error extracting Python zip file:", err)
			return
		}

		err = common.DecompressIOStream(PayloadReader, settings.ScriptExtractDir)
		if err != nil {
			fmt.Println("Error extracting payload zip file:", err)
			return
		}

		wheelsDir := path.Join(settings.PythonExtractDir, common.WheelsFilename)

		err = common.DecompressIOStream(wheelsReader, wheelsDir)
		if err != nil {
			fmt.Println("Error extracting wheels zip file:", err)
			return
		}

		pythonPath := filepath.Join(settings.PythonExtractDir, "python.exe")

		if err := common.RunCommand(pythonPath, []string{common.GetPipName(settings.PythonExtractDir), "install", "pip", "setuptools", "wheel"}); err != nil {
			fmt.Println("Error building wheels:", err)
			return
		}

		// if requirements.txt exists, install the requirements
		if _, err := os.Stat(settings.RequirementsFile); err == nil {
			if err := common.RunCommand(pythonPath, []string{common.GetPipName(settings.PythonExtractDir), "install", "--find-links", path.Join(wheelsDir) + "/", "--only-binary=:all:", "-r", settings.RequirementsFile}); err != nil {
				fmt.Println("Error while installing requirements from disk... Continuing...", err)
			}
		}

		// setup script path is relative to the extracted script directory
		setupScriptPath := path.Join(settings.ScriptExtractDir, settings.SetupScript)

		// run the setup.py file if configured
		if settings.SetupScript != "" {
			if err := common.RunCommand(pythonPath, []string{setupScriptPath}); err != nil {
				fmt.Println("Error running "+settings.SetupScript+":", err)
				return
			}
		}

		myHash, err := calculateSelfHash()

		err = common.SaveContentsToFile("bootstrapped", myHash)
		if err != nil {
			fmt.Println("Error saving hash to file:", err)
		}

	}

	//TODO: Add some form of hashing of the input files; E.g., make a list of all files present when making the installer and hash them
	// Save the list of files and their hashes to a file in the installer
	// When the installer is run, hash the same files and compare them to the saved hashes

	EmbeddedIntegrityHashes := attachments.Reader(common.IntegrityFilename)

	if EmbeddedIntegrityHashes == nil {
		panic("Error reading integrity hashes. Ensure they are embedded in the binary.")
	}

	integrityData, err := io.ReadAll(EmbeddedIntegrityHashes)
	if err != nil {
		panic("Error reading data from reader: " + err.Error())
	}

	// these will be in the form of a json string, so we need to unmarshal them
	var fileHashes []common.FileHash

	// Unmarshal JSON string to slice of FileHash objects
	err = json.Unmarshal(integrityData, &fileHashes)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	// get the hashes of the extracted files
	tampered, err := common.VerifyDirectoryHashes(settings.ScriptExtractDir, fileHashes)

	if err != nil {
		panic(err)
	}

	if len(tampered) > 0 {

		fmt.Println("Error validating integrity of extracted files.")
		fmt.Println("Warning, the following files have been modified since installation:")

		for _, file := range tampered {
			fmt.Println(file)
		}

		if pure {
			fmt.Println("Please re-run the installer.")
			os.Remove("bootstrapped")

			// quit the program with an error code
			os.Exit(1000)
		}
	} else {
		fmt.Println("Installation integrity validated successfully.")
	}

	attachments.Close()

	// run the payload script

	pythonExecutable := filepath.Join(settings.PythonExtractDir, "python.exe")
	mainScriptPath := path.Join(settings.ScriptExtractDir, settings.MainScript)

	if !pure {
		fmt.Println("Running script...")

		if err := common.RunScript(pythonExecutable, mainScriptPath, settings.ScriptExtractDir, os.Args[1:]); err != nil {
			fmt.Println("Error running Python script:", err)
			return
		}

		fmt.Println("Script completed.")
	} else {

		// replace the placeholders in the runscript with the actual values
		runScript = strings.ReplaceAll(runScript, "{{PYTHON_EXE}}", pythonExecutable)
		runScript = strings.ReplaceAll(runScript, "{{MAIN_SCRIPT}}", mainScriptPath)
		runScript = strings.ReplaceAll(runScript, "{{SCRIPTS_DIR}}", settings.ScriptExtractDir)

		err = os.WriteFile("run.bat", []byte(runScript), 0644)

		// get path to run.bat
		runBatPath, err := filepath.Abs("run.bat")
		if err != nil {
			fmt.Println("Error getting absolute path for run.bat:", err)
			return
		}

		fmt.Println("Please run the following command in the command line to run the script:")
		fmt.Println(runBatPath)

	}

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

			PressButtonToContinue("Press enter to accept the new hash and continue...")

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
		fmt.Println("certutil -hashfile", os.Args[0], "MD5")
		fmt.Println("Note: If hash values do not match, the file may have been tampered with.")

		PressButtonToContinue("Press enter to continue...")
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

func PressButtonToContinue(continueMessage string) {
	fmt.Println(continueMessage)
	fmt.Println(".")
	fmt.Print("\a")

	stop := make(chan bool)

	go func() {
		animation := []string{" ", " ", " ", "o", "O", "o", " ", " ", " "}
		i := 0
		for {
			select {
			case <-stop:
				fmt.Printf("\r%s", strings.Repeat(" ", len(strings.Join(animation, ""))))
				return
			default:
				fmt.Printf("\r%s", strings.Join(animation, ""))
				time.Sleep(100 * time.Millisecond)
				animation = append(animation[1:], animation[0])
				i++
				if i == len(animation) {
					i = 0
					animation = []string{" ", " ", " ", "o", "O", "o", " ", " ", " "}
				}
			}
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')

	stop <- true
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
