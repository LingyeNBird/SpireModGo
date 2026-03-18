package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var backupSlotRE = regexp.MustCompile(`_p(\d)_`)

func (m *Manager) GetSaveSlotInfo(steamID string, saveType SaveType, slot int) (SaveSlotInfo, error) {
	path := m.saveSlotPath(steamID, saveType, slot)
	info := SaveSlotInfo{Type: saveType, Slot: slot, Path: path}
	progressPath := filepath.Join(path, "progress.save")
	progressInfo, err := os.Stat(progressPath)
	if err == nil {
		info.HasData = true
		info.LastModified = progressInfo.ModTime()
	}
	if fileExists(filepath.Join(path, "current_run.save")) {
		info.HasCurrentRun = true
	}
	return info, nil
}

func (m *Manager) ListSaveSlots(steamID string, saveType SaveType) ([]SaveSlotInfo, error) {
	slots := make([]SaveSlotInfo, 0, 3)
	for slot := 1; slot <= 3; slot++ {
		info, err := m.GetSaveSlotInfo(steamID, saveType, slot)
		if err != nil {
			return nil, err
		}
		slots = append(slots, info)
	}
	return slots, nil
}

func (m *Manager) CopySave(steamID string, srcType SaveType, srcSlot int, dstType SaveType, dstSlot int, options SaveCopyOptions) (SaveCopyResult, error) {
	if options.BackupTag == "" {
		options.BackupTag = "auto_before_copy"
	}
	srcInfo, err := m.GetSaveSlotInfo(steamID, srcType, srcSlot)
	if err != nil {
		return SaveCopyResult{}, err
	}
	if !srcInfo.HasData {
		return SaveCopyResult{}, fmt.Errorf("source slot %s%d is empty", saveTypePrefix(srcType), srcSlot)
	}
	dstInfo, err := m.GetSaveSlotInfo(steamID, dstType, dstSlot)
	if err != nil {
		return SaveCopyResult{}, err
	}

	result := SaveCopyResult{}
	if dstInfo.HasData {
		backupDir, err := m.BackupSave(steamID, dstType, options.BackupTag, dstSlot)
		if err != nil {
			return result, err
		}
		result.BackupDir = backupDir
	}
	if err := ensureDir(dstInfo.Path); err != nil {
		return result, err
	}

	entries, err := os.ReadDir(srcInfo.Path)
	if err != nil {
		return result, err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(srcInfo.Path, entry.Name())
		dstPath := filepath.Join(dstInfo.Path, entry.Name())
		if entry.IsDir() {
			count, err := copyDirRecursive(srcPath, dstPath)
			if err != nil {
				return result, err
			}
			result.CopiedFiles += count
			continue
		}
		if options.CreateBeforeCopyBackup && fileExists(dstPath) {
			_ = copyRegularFile(dstPath, dstPath+".before_copy")
		}
		if err := copyRegularFile(srcPath, dstPath); err != nil {
			return result, err
		}
		result.CopiedFiles++
	}
	if result.CopiedFiles > 0 {
		cloudSynced, cloudUpdated, err := m.SyncSteamCloudCache(steamID, dstType, dstSlot, dstInfo.Path)
		if err != nil {
			m.logf("cloud sync failed: %v", err)
		} else {
			result.CloudSynced = cloudSynced
			result.CloudUpdated = cloudUpdated
		}
	}
	return result, nil
}

func (m *Manager) BackupSave(steamID string, saveType SaveType, tag string, slot int) (string, error) {
	if slot == 0 {
		slot = 1
	}
	if tag == "" {
		tag = "manual"
	}
	srcPath := m.saveSlotPath(steamID, saveType, slot)
	if !dirExists(srcPath) {
		return "", fmt.Errorf("save directory does not exist: %s", srcPath)
	}
	timestamp := time.Now().Format("20060102_150405")
	backupDir := filepath.Join(m.SaveRoot, steamID, "backups", fmt.Sprintf("%s_p%d_%s_%s", saveType, slot, tag, timestamp))
	if err := ensureDir(backupDir); err != nil {
		return "", err
	}

	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return "", err
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.Contains(strings.ToLower(entry.Name()), ".save") {
			continue
		}
		if err := copyRegularFile(filepath.Join(srcPath, entry.Name()), filepath.Join(backupDir, entry.Name())); err != nil {
			return "", err
		}
		count++
	}
	if count == 0 {
		_ = os.RemoveAll(backupDir)
		return "", nil
	}
	if err := m.cleanupOldBackups(steamID, saveType); err != nil {
		m.logf("backup cleanup failed: %v", err)
	}
	return backupDir, nil
}

func (m *Manager) cleanupOldBackups(steamID string, saveType SaveType) error {
	backupRoot := filepath.Join(m.SaveRoot, steamID, "backups")
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var names []string
	prefix := string(saveType) + "_"
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			names = append(names, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	for idx, name := range names {
		if idx < 20 {
			continue
		}
		_ = os.RemoveAll(filepath.Join(backupRoot, name))
	}
	return nil
}

func (m *Manager) ListBackups(steamID string) ([]BackupInfo, error) {
	backupRoot := filepath.Join(m.SaveRoot, steamID, "backups")
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	backups := make([]BackupInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		backup := BackupInfo{Name: entry.Name(), FullPath: filepath.Join(backupRoot, entry.Name())}
		if strings.HasPrefix(entry.Name(), "normal_") {
			backup.Type = SaveTypeNormal
		} else {
			backup.Type = SaveTypeModded
		}
		if match := backupSlotRE.FindStringSubmatch(entry.Name()); len(match) == 2 {
			fmt.Sscanf(match[1], "%d", &backup.Slot)
		}
		files, _ := os.ReadDir(backup.FullPath)
		for _, file := range files {
			if !file.IsDir() {
				backup.FileCount++
			}
		}
		backups = append(backups, backup)
	}
	sort.Slice(backups, func(i, j int) bool { return backups[i].Name > backups[j].Name })
	return backups, nil
}

func (m *Manager) RestoreBackup(steamID string, backup BackupInfo, targetSlot int) (SaveCopyResult, error) {
	if targetSlot == 0 {
		targetSlot = backup.Slot
	}
	result := SaveCopyResult{}
	targetPath := m.saveSlotPath(steamID, backup.Type, targetSlot)
	if fileExists(filepath.Join(targetPath, "progress.save")) {
		backupDir, err := m.BackupSave(steamID, backup.Type, "auto_before_restore", targetSlot)
		if err != nil {
			return result, err
		}
		result.BackupDir = backupDir
	}
	if err := ensureDir(targetPath); err != nil {
		return result, err
	}
	entries, err := os.ReadDir(backup.FullPath)
	if err != nil {
		return result, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if err := copyRegularFile(filepath.Join(backup.FullPath, entry.Name()), filepath.Join(targetPath, entry.Name())); err != nil {
			return result, err
		}
		result.CopiedFiles++
	}
	cloudSynced, cloudUpdated, err := m.SyncSteamCloudCache(steamID, backup.Type, targetSlot, targetPath)
	if err != nil {
		m.logf("cloud sync failed after restore: %v", err)
	} else {
		result.CloudSynced = cloudSynced
		result.CloudUpdated = cloudUpdated
	}
	return result, nil
}

func (m *Manager) saveSlotPath(steamID string, saveType SaveType, slot int) string {
	if saveType == SaveTypeNormal {
		return filepath.Join(m.SaveRoot, steamID, fmt.Sprintf("profile%d", slot), "saves")
	}
	return filepath.Join(m.SaveRoot, steamID, "modded", fmt.Sprintf("profile%d", slot), "saves")
}

func saveTypePrefix(saveType SaveType) string {
	if saveType == SaveTypeNormal {
		return "A"
	}
	return "B"
}
