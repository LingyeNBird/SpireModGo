//go:build windows

package dialog

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func PickZipImportFile(startDir string) (string, error) {
	path, err := runPowerShellFileDialog(`Add-Type -AssemblyName System.Windows.Forms; $dialog = New-Object System.Windows.Forms.OpenFileDialog; $dialog.Filter = 'ZIP archive (*.zip)|*.zip'; if ($env:STS2_DIALOG_DIR) { $dialog.InitialDirectory = $env:STS2_DIALOG_DIR }; if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { [Console]::OutputEncoding = [System.Text.Encoding]::UTF8; Write-Output $dialog.FileName; exit 0 }; exit 3`, startDir, "")
	if err != nil {
		return "", err
	}
	return path, nil
}

func PickZipExportFile(startDir, defaultName string) (string, error) {
	path, err := runPowerShellFileDialog(`Add-Type -AssemblyName System.Windows.Forms; $dialog = New-Object System.Windows.Forms.SaveFileDialog; $dialog.Filter = 'ZIP archive (*.zip)|*.zip'; $dialog.DefaultExt = 'zip'; $dialog.AddExtension = $true; $dialog.OverwritePrompt = $true; if ($env:STS2_DIALOG_DIR) { $dialog.InitialDirectory = $env:STS2_DIALOG_DIR }; if ($env:STS2_DIALOG_FILE) { $dialog.FileName = $env:STS2_DIALOG_FILE }; if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { [Console]::OutputEncoding = [System.Text.Encoding]::UTF8; Write-Output $dialog.FileName; exit 0 }; exit 3`, startDir, defaultName)
	if err != nil {
		return "", err
	}
	if filepath.Ext(path) == "" {
		path += ".zip"
	}
	return path, nil
}

func runPowerShellFileDialog(script, startDir, startFile string) (string, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-STA", "-Command", script)
	cmd.Env = append(os.Environ(),
		"STS2_DIALOG_DIR="+startDir,
		"STS2_DIALOG_FILE="+startFile,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 3 {
			return "", ErrCancelled
		}
		return "", fmt.Errorf("run file dialog: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}
