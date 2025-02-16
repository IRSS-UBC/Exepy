package main

import (
	"fmt"
	"io"
	common "lukasolson.net/common"
	"os"
	"path/filepath"
)

func PreparePython(settings common.PythonSetupSettings) (io.ReadSeeker, io.ReadSeeker, error) {

	cleanDirectory(&settings)

	defer cleanDirectory(&settings)

	// CREATE THE EXTRACTION DIRECTORY
	common.RemoveIfExists(settings.PythonExtractDir)
	if err := os.Mkdir(settings.PythonExtractDir, os.ModePerm); err != nil {
		fmt.Println("Error creating extraction directory:", err)
		return nil, nil, err
	}

	// DOWNLOAD PYTHON ZIP FILE
	if err := common.DownloadFile(settings.PythonDownloadURL, settings.PythonDownloadZip); err != nil {
		fmt.Println("Error downloading Python zip file:", err)
		return nil, nil, err
	}

	// DOWNLOAD PIP FILE
	if err := common.DownloadFile(settings.PipDownloadURL, common.GetPipName(settings.PythonExtractDir)); err != nil {
		fmt.Println("Error downloading pip module:", err)
		return nil, nil, err
	}

	if err := createBasePythonInstallation(&settings, settings.PythonDownloadZip); err != nil {
		fmt.Println("Error creating base Python installation:", err)
		return nil, nil, err
	}

	common.RemoveIfExists(settings.PythonDownloadZip)

	originRequirements := filepath.Join(settings.ScriptDir, settings.RequirementsFile)
	destRequirements := filepath.Join(settings.PythonExtractDir, settings.RequirementsFile)
	common.CopyFile(originRequirements, destRequirements)

	pythonStream, err := common.DirToStream(settings.PythonExtractDir, []string{})

	if err != nil {
		fmt.Println("Error zipping Python directory:", err)
		return nil, nil, err
	}

	wheelsPath := filepath.Join(settings.PythonExtractDir, "wheels")
	os.Mkdir(wheelsPath, os.ModePerm)

	if settings.RequirementsFile != "" {

		if common.DoesPathExist(originRequirements) {
			fmt.Println("Requirements file found:", originRequirements)
			if err := buildRequirementWheels(settings.PythonExtractDir, originRequirements, wheelsPath); err != nil {
				return nil, nil, err
			}
		} else {
			fmt.Println("Requirements file not found but is specified in configuration:", originRequirements)
		}

	}

	wheelsStream, _ := common.DirToStream(wheelsPath, []string{})

	return pythonStream, wheelsStream, nil
}

func createBasePythonInstallation(settings *common.PythonSetupSettings, pythonZip string) error {
	// EXTRACT THE Python ZIP FILE
	if err := common.ExtractZip(pythonZip, settings.PythonExtractDir, 0); err != nil {
		fmt.Println("Error extracting Python zip file:", err)
		return err
	}

	if err := extractInteriorPythonArchive(settings); err != nil {
		return err
	}

	if err := updatePTHFile(settings); err != nil {
		return err
	}

	// write to sitecustomize.py file
	if err := createSiteCustomFile(settings); err != nil {
		return err
	}

	// make empty DLLs folder
	if err := os.Mkdir(filepath.Join(settings.PythonExtractDir, "DLLs"), os.ModePerm); err != nil {
		fmt.Println("Error creating DLLs folder:", err)
		return err
	}

	return nil
}

func extractInteriorPythonArchive(settings *common.PythonSetupSettings) error {
	// EXTRACT THE EMBEDDED PYTHON INTERIOR ZIP FILE

	if err := common.ExtractZip(filepath.Join(settings.PythonExtractDir, settings.PythonInteriorZip), settings.PythonExtractDir, 0); err != nil {
		fmt.Println("Error extracting the interiorPython zip file:", err)
		return err
	}

	common.RemoveIfExists(filepath.Join(settings.PythonExtractDir, settings.PythonInteriorZip))
	return nil
}

func createSiteCustomFile(settings *common.PythonSetupSettings) error {
	sitecustomizeFile, err := os.Create(filepath.Join(settings.PythonExtractDir, "sitecustomize.py"))

	if err != nil {
		return err
	}

	_, err = sitecustomizeFile.WriteString("import sys\nsys.path.append('.')")

	if err != nil {
		return err
	}

	return nil
}

func updatePTHFile(settings *common.PythonSetupSettings) error {
	common.RemoveIfExists(filepath.Join(settings.PythonExtractDir, settings.PthFile))

	// write to ._pth file
	pthFile, err := os.Create(filepath.Join(settings.PythonExtractDir, settings.PthFile))
	if err != nil {
		fmt.Println("Error creating ._pth file:", err)
		return nil
	}

	// change python311 to pythonExtractDir
	_, err = pthFile.WriteString(".\\" + settings.PythonExtractDir + "\n.\\Scripts\n.\n.\\Lib\\site-packages\nimport site")
	if err != nil {
		fmt.Println("Error writing to ._pth file:", err)
		return nil
	}
	return err
}

func buildRequirementWheels(extractDir, requirementsFile, wheelDir string) error {

	pythonPath := filepath.Join(extractDir, "python.exe")

	if err := common.RunCommand(pythonPath, []string{common.GetPipName(extractDir), "install", "pip", "setuptools", "wheel"}); err != nil {
		fmt.Println("Error building wheels:", err)
		return err
	}

	if err := common.RunCommand(pythonPath, []string{common.GetPipName(extractDir), "wheel", "-w", wheelDir, "-r", requirementsFile}); err != nil {
		fmt.Println("Error building wheels:", err)
		return err
	}

	return nil
}

func cleanDirectory(settings *common.PythonSetupSettings) {
	common.RemoveIfExists(settings.PythonExtractDir)
	common.RemoveIfExists(settings.PythonDownloadZip)

	println("Directory cleaned")
}
