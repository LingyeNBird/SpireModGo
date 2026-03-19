//go:build !windows

package ui

import "fmt"

func pickZipImportFile(startDir string) (string, error) {
	return "", fmt.Errorf("zip import dialog is only supported on Windows")
}

func pickZipExportFile(startDir, defaultName string) (string, error) {
	return "", fmt.Errorf("zip export dialog is only supported on Windows")
}
