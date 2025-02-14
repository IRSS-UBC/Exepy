package common

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RunCommand(command string, args []string) error {
	cmd, err := createCommand(command, args)
	if err != nil {
		return err
	}

	println("Running command:", cmd.String())
	return cmd.Run()
}

func RunScript(pythonExecutablePath string, mainScriptPath string, scriptsDir string, args []string) error {

	cmd, err := createCommand(pythonExecutablePath, args)
	if err != nil {
		return err
	}

	// Get the absolute path of the scripts directory
	absScriptsDir, err := filepath.Abs(scriptsDir)
	if err != nil {
		return fmt.Errorf("error determining absolute path for scripts directory: %v", err)
	}

	// Get the absolute path of the main script
	absMainScript, err := filepath.Abs(mainScriptPath)
	if err != nil {
		return fmt.Errorf("error determining absolute path for main script: %v", err)
	}

	// Construct the Python code for the -c flag
	pythonCode := fmt.Sprintf(
		"import sys; sys.path.insert(0, '%s'); exec(open('%s').read())",
		strings.ReplaceAll(absScriptsDir, "\\", "/"),
		strings.ReplaceAll(absMainScript, "\\", "/"),
	)

	cmd.Args = append(cmd.Args, "-c", pythonCode)

	// Properly set the environment variable.
	cmd.Env = append(os.Environ(), fmt.Sprintf("PYTHONPATH=%s", absScriptsDir))

	println("Running script:", cmd.String())
	return cmd.Run()
}

func createCommand(command string, args []string) (*exec.Cmd, error) {
	cmd := exec.Command(command, args...)

	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("error getting executable path: %v", err)
	}

	exeDir := filepath.Dir(execPath)
	cmd.Dir = exeDir

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
}
