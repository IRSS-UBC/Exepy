package common

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func RunCommand(command string, args []string) error {
	cmd, err := createCommand(command, args)
	if err != nil {
		return err
	}

	println("Running command:", cmd.String())
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
