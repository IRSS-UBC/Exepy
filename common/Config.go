package common

import (
	"encoding/json"
	"errors"
	"io/ioutil"
)

type PythonSetupSettings struct {
	PythonDownloadURL string `json:"pythonDownloadURL"`
	PipDownloadURL    string `json:"pipDownloadURL"`
	PythonDownloadZip string `json:"pythonDownloadFile"`
	PythonExtractDir  string `json:"pythonExtractDir"`
	ScriptExtractDir  string `json:"scriptExtractDir"`
	PthFile           string `json:"pthFile"`
	PythonInteriorZip string `json:"pythonInteriorZip"`
	RequirementsFile  string `json:"requirementsFile"`
	ScriptDir         string `json:"scriptDir"`
	SetupScript       string `json:"setupScript"`
	MainScript        string `json:"mainScript"`
}

func loadSettings(filename string) (*PythonSetupSettings, error) {
	data, err := ioutil.ReadFile(filename)
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
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func LoadOrSaveDefault(filename string) (*PythonSetupSettings, error) {
	settings, err := loadSettings(filename)
	if err != nil {
		settings = &PythonSetupSettings{
			PythonDownloadURL: "",
			PipDownloadURL:    "",
			PythonDownloadZip: "",
			PythonExtractDir:  "",
			ScriptExtractDir:  "scripts",
			PthFile:           "",
			PythonInteriorZip: "",
			ScriptDir:         "scripts",
			RequirementsFile:  "",
			MainScript:        "",
		}

		err = saveSettings(filename, settings)
		if err != nil {
			return nil, err
		}

		if settings.MainScript == "" {
			return nil, errors.New("mainScript is required in settings.json. Please add it and try again")
		}
	}

	return settings, nil
}
