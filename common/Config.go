package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// PythonSetupSettings holds the configuration settings.
type PythonSetupSettings struct {
	PythonDownloadURL     *string  `json:"pythonDownloadURL"`
	PipDownloadURL        *string  `json:"pipDownloadURL"`
	PythonDownloadZip     *string  `json:"pythonDownloadFile"`
	PythonExtractDir      *string  `json:"pythonExtractDir"`
	ScriptExtractDir      *string  `json:"scriptExtractDir"`
	PthFile               *string  `json:"pthFile"`
	PythonInteriorZip     *string  `json:"pythonInteriorZip"`
	InstallerRequirements *string  `json:"installerRequirements"`
	RequirementsFile      *string  `json:"requirementsFile"`
	ScriptDir             *string  `json:"scriptDir"`
	SetupScript           *string  `json:"setupScript"`
	MainScript            *string  `json:"mainScript"`
	FilesToCopyToRoot     []string `json:"filesToCopyToRoot"`
	RunAfterInstall       *bool    `json:"runAfterInstall"`
}

// Validate checks if the required fields are present.
func (s *PythonSetupSettings) Validate() (err error) {
	// Recover from any unexpected panics
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("validation error: %v", r)
		}
	}()

	// Check each pointer field before de-referencing it.
	if s.PythonDownloadURL == nil {
		return errors.New("missing required field: pythonDownloadURL")
	}
	if *s.PythonDownloadURL == "" {
		return errors.New("required field is empty: pythonDownloadURL")
	}

	if s.PipDownloadURL == nil {
		return errors.New("missing required field: pipDownloadURL")
	}
	if *s.PipDownloadURL == "" {
		return errors.New("required field is empty: pipDownloadURL")
	}

	if s.PythonDownloadZip == nil {
		return errors.New("missing required field: pythonDownloadFile")
	}
	if s.PythonExtractDir == nil {
		return errors.New("missing required field: pythonExtractDir")
	}
	if s.ScriptExtractDir == nil {
		return errors.New("missing required field: scriptExtractDir")
	}
	if s.PthFile == nil {
		return errors.New("missing required field: pthFile")
	}
	if s.PythonInteriorZip == nil {
		return errors.New("missing required field: pythonInteriorZip")
	}
	if s.ScriptDir == nil {
		return errors.New("missing required field: scriptDir")
	}
	if s.SetupScript == nil {
		return errors.New("missing required field: setupScript")
	}
	if s.MainScript == nil {
		return errors.New("missing required field: mainScript")
	}

	// If all validations pass, return nil
	return nil
}

// loadSettings reads the configuration file and unmarshals it.
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

// saveSettings writes the configuration to the file.
func saveSettings(filename string, settings *PythonSetupSettings) error {
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

func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

// mergeSettings fills in any nil fields in loaded with defaults.
func mergeSettings(loaded, defaults *PythonSetupSettings) *PythonSetupSettings {
	if loaded.PythonDownloadURL == nil {
		loaded.PythonDownloadURL = defaults.PythonDownloadURL
	}
	if loaded.PipDownloadURL == nil {
		loaded.PipDownloadURL = defaults.PipDownloadURL
	}
	if loaded.PythonDownloadZip == nil {
		loaded.PythonDownloadZip = defaults.PythonDownloadZip
	}
	if loaded.PythonExtractDir == nil {
		loaded.PythonExtractDir = defaults.PythonExtractDir
	}
	if loaded.ScriptExtractDir == nil {
		loaded.ScriptExtractDir = defaults.ScriptExtractDir
	}
	if loaded.PthFile == nil {
		loaded.PthFile = defaults.PthFile
	}
	if loaded.PythonInteriorZip == nil {
		loaded.PythonInteriorZip = defaults.PythonInteriorZip
	}
	if loaded.InstallerRequirements == nil {
		loaded.InstallerRequirements = defaults.InstallerRequirements
	}
	if loaded.RequirementsFile == nil {
		loaded.RequirementsFile = defaults.RequirementsFile
	}
	if loaded.ScriptDir == nil {
		loaded.ScriptDir = defaults.ScriptDir
	}
	if loaded.SetupScript == nil {
		loaded.SetupScript = defaults.SetupScript
	}
	if loaded.MainScript == nil {
		loaded.MainScript = defaults.MainScript
	}

	if loaded.RunAfterInstall == nil {
		loaded.RunAfterInstall = defaults.RunAfterInstall
	}

	// For the slice, merge if it's empty.
	if loaded.FilesToCopyToRoot == nil || len(loaded.FilesToCopyToRoot) == 0 {
		loaded.FilesToCopyToRoot = defaults.FilesToCopyToRoot
	}
	// RunAfterInstall is a bool; false is a valid default.
	return loaded
}

// LoadOrSaveDefault loads settings from the file or creates a default configuration if necessary.
func LoadOrSaveDefault(filename string) (*PythonSetupSettings, error) {
	// Define default settings.
	defaults := &PythonSetupSettings{
		PythonDownloadURL:     strPtr(""),
		PipDownloadURL:        strPtr(""),
		PythonDownloadZip:     strPtr("python code-3.11.7-embed-amd64.zip"),
		PythonExtractDir:      strPtr("python-embed"),
		ScriptExtractDir:      strPtr("scripts"),
		PthFile:               strPtr("python311._pth"),
		PythonInteriorZip:     strPtr("python311.zip"),
		InstallerRequirements: strPtr(""),
		RequirementsFile:      strPtr("requirements.txt"),
		ScriptDir:             strPtr("scripts"),
		SetupScript:           strPtr(""),
		MainScript:            strPtr("main.py"),
		FilesToCopyToRoot:     []string{"requirements.txt", "readme.md", "license.md"},
		RunAfterInstall:       boolPtr(false),
	}

	// Attempt to load the existing configuration.
	loaded, err := loadSettings(filename)
	if err != nil {
		// If the file doesn't exist or can't be read, save defaults.
		if saveErr := saveSettings(filename, defaults); saveErr != nil {
			return nil, saveErr
		}
		return defaults, nil
	}

	merged := mergeSettings(loaded, defaults)

	// Validate the merged configuration.
	if err := merged.Validate(); err != nil {
		return nil, err
	}

	// Check for incomplete settings.

	// Save the merged configuration back to the file.
	if err := saveSettings(filename, merged); err != nil {
		return nil, err
	}

	return merged, nil
}
