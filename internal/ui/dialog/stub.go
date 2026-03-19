//go:build !windows

package dialog

import "fmt"

func PickZipImportFile(startDir string) (string, error) {
	return "", fmt.Errorf("zip import dialog is only supported on Windows")
}

func PickZipExportFile(startDir, defaultName string) (string, error) {
	return "", fmt.Errorf("zip export dialog is only supported on Windows")
}
