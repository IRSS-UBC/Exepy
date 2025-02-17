package main

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"
)

var (
	modpsapi                     = windows.NewLazySystemDLL("psapi.dll")
	procGetProcessImageFileNameW = modpsapi.NewProc("GetProcessImageFileNameW")
)

// GetProcessImageFileName calls the Windows API function from psapi.dll.
// It fills the buffer pointed to by lpImageFileName with the process image file name.
func GetProcessImageFileName(handle windows.Handle, lpImageFileName *uint16, nSize uint32) (uint32, error) {
	ret, _, err := procGetProcessImageFileNameW.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(lpImageFileName)),
		uintptr(nSize),
	)
	if ret == 0 {
		if err != windows.ERROR_SUCCESS {
			return 0, err
		}
		return 0, fmt.Errorf("GetProcessImageFileNameW failed")
	}
	return uint32(ret), nil
}

// isLaunchedFromExplorer checks if the parent process is explorer.exe.
func isLaunchedFromExplorer() bool {
	if runtime.GOOS != "windows" {
		return false
	}

	ppid := os.Getppid()

	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, uint32(ppid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)

	// Allocate a buffer for the process image file name (UTF-16).
	var processName [260]uint16

	n, err := GetProcessImageFileName(handle, &processName[0], uint32(len(processName)))
	if err != nil || n == 0 {
		return false
	}

	// Convert the UTF-16 buffer to a Go string.
	processNameStr := windows.UTF16ToString(processName[:n])

	baseName := filepath.Base(processNameStr)
	return strings.EqualFold(baseName, "explorer.exe")
}
