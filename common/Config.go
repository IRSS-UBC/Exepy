package common

import (
	"encoding/json"
	"errors"
	"os"
)

type PythonSetupSettings struct {
	PythonDownloadURL     string   `json:"pythonDownloadURL"`
	PipDownloadURL        string   `json:"pipDownloadURL"`
	PythonDownloadZip     string   `json:"pythonDownloadFile"`
	PythonExtractDir      string   `json:"pythonExtractDir"`
	ScriptExtractDir      string   `json:"scriptExtractDir"`
	PthFile               string   `json:"pthFile"`
	PythonInteriorZip     string   `json:"pythonInteriorZip"`
	InstallerRequirements string   `json:"installerRequirements"` // This is the requirements file that will be used to build the wheels for the installer. It is not included in the installer.
	RequirementsFile      string   `json:"requirementsFile"`      // This is the requirements file that will be used at install-time to install the wheels.
	ScriptDir             string   `json:"scriptDir"`
	SetupScript           string   `json:"setupScript"`
	MainScript            string   `json:"mainScript"`
	FilesToCopyToRoot     []string `json:"filesToCopyToRoot"`
	RunAfterInstall       bool     `json:"runAfterInstall"`
}

func loadSettings(filename string) (*PythonSetupSettings, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var settings PythonSetupSettings
	err = json.Unmarshal(data, &settings)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

func saveSettings(filename string, settings *PythonSetupSettings) error {
	// check if the file exists

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func LoadOrSaveDefault(filename string) (*PythonSetupSettings, error) {
	settings, err := loadSettings(filename)
	if err != nil {
		settings = &PythonSetupSettings{
			PythonDownloadURL:     "",
			PipDownloadURL:        "",
			PythonDownloadZip:     "python code-3.11.7-embed-amd64.zip",
			PythonExtractDir:      "python-embed",
			ScriptExtractDir:      "scripts",
			PthFile:               "python311._pth",
			PythonInteriorZip:     "python311.zip",
			InstallerRequirements: "",
			RequirementsFile:      "requirements.txt",
			ScriptDir:             "scripts",
			MainScript:            "main.py",
			FilesToCopyToRoot:     []string{"requirements.txt", "readme.md", "license.md"},
			RunAfterInstall:       false,
		}

		err = saveSettings(filename, settings)
		if err != nil {
			return nil, err
		}

		if settings.MainScript == "" {
			return nil, errors.New("mainScript is required in " + filename + ". Please add it and try again")
		}
	}

	return settings, nil
}
