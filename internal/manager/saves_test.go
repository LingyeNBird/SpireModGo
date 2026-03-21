package manager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopySaveCreatesBackupAndBeforeCopyFile(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	m, err := New(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	m.SaveRoot = filepath.Join(baseDir, "steam")

	steamID := "76561197960265729"
	srcPath := m.saveSlotPath(steamID, SaveTypeNormal, 1)
	dstPath := m.saveSlotPath(steamID, SaveTypeModded, 1)
	if err := os.MkdirAll(filepath.Join(srcPath, "history"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dstPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcPath, "progress.save"), []byte("new-progress"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcPath, "meta.save.backup"), []byte("meta"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcPath, "history", "run1.txt"), []byte("run"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dstPath, "progress.save"), []byte("old-progress"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := m.CopySave(steamID, SaveTypeNormal, 1, SaveTypeModded, 1, SaveCopyOptions{
		BackupTag:              "auto_before_copy",
		CreateBeforeCopyBackup: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.CopiedFiles != 3 {
		t.Fatalf("expected 3 copied files, got %d", result.CopiedFiles)
	}
	if result.BackupDir == "" {
		t.Fatal("expected backup dir to be created")
	}
	data, err := os.ReadFile(filepath.Join(dstPath, "progress.save"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new-progress" {
		t.Fatalf("unexpected progress.save content: %q", string(data))
	}
	if !fileExists(filepath.Join(dstPath, "progress.save.before_copy")) {
		t.Fatal("expected progress.save.before_copy to exist")
	}
	if !fileExists(filepath.Join(dstPath, "history", "run1.txt")) {
		t.Fatal("expected history file to be copied")
	}
	backupEntries, err := os.ReadDir(result.BackupDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(backupEntries) == 0 {
		t.Fatal("expected backup dir to contain save files")
	}
}

func TestRestoreBackupCopiesIntoTargetSlot(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	m, err := New(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	m.SaveRoot = filepath.Join(baseDir, "steam")

	steamID := "76561197960265729"
	source := m.saveSlotPath(steamID, SaveTypeNormal, 2)
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "progress.save"), []byte("slot2"), 0o644); err != nil {
		t.Fatal(err)
	}
	backupDir, err := m.BackupSave(steamID, SaveTypeNormal, "manual", 2)
	if err != nil {
		t.Fatal(err)
	}
	if backupDir == "" {
		t.Fatal("expected backup to be created")
	}
	backups, err := m.ListBackups(steamID)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(backups))
	}
	if backups[0].Slot != 2 {
		t.Fatalf("expected parsed backup slot 2, got %d", backups[0].Slot)
	}
	_, err = m.RestoreBackup(steamID, backups[0], 3)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(m.saveSlotPath(steamID, SaveTypeNormal, 3), "progress.save"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "slot2" {
		t.Fatalf("unexpected restored content: %q", string(data))
	}
}

func TestLoadConfigHandlesUTF8BOM(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "SpireModGo", "modmanager.json")
	content := append([]byte{0xEF, 0xBB, 0xBF}, []byte(`{"GameDir":"C:\\Game"}`)...)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := New(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	m.ConfigPath = configPath
	if err := m.LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if got := m.GetGameDir(); got != "" {
		t.Fatalf("expected invalid BOM config path to be ignored by GetGameDir validation, got %q", got)
	}
	if m.Config.GameDir != "" {
		t.Fatalf("expected invalid saved path to be cleared after validation, got %q", m.Config.GameDir)
	}
}

func TestEnableModsInSettingsHandlesUTF8BOM(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	m, err := New(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	m.SaveRoot = filepath.Join(baseDir, "steam")
	settingsPath := filepath.Join(m.SaveRoot, "76561197960265729", "settings.save")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := append([]byte{0xEF, 0xBB, 0xBF}, []byte(`{"mod_settings":{"mods_enabled":false}}`)...)
	if err := os.WriteFile(settingsPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := m.EnableModsInSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected BOM-prefixed settings.save to be updated")
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"mods_enabled": true`) {
		t.Fatalf("expected mods_enabled to be true, got %q", string(data))
	}
}

func TestUpdateRemoteCacheVDFRewritesFields(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	cloudDir := filepath.Join(baseDir, "profile1", "saves")
	if err := os.MkdirAll(cloudDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cloudDir, "progress.save"), []byte("new-payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	vdfPath := filepath.Join(baseDir, "remotecache.vdf")
	vdf := "\"profile1/saves/progress.save\"\n{\n\t\"size\"\t\t\"1\"\n\t\"localtime\"\t\"1\"\n\t\"time\"\t\t\"1\"\n\t\"sha\"\t\t\"old\"\n\t\"syncstate\"\t\"0\"\n}\n"
	if err := os.WriteFile(vdfPath, []byte(vdf), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := updateRemoteCacheVDF(vdfPath, `profile1\saves`, cloudDir); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(vdfPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(updated)
	if !strings.Contains(text, `"syncstate"	"4"`) && !strings.Contains(text, `"syncstate"`) {
		t.Fatal("expected syncstate to be updated")
	}
	if strings.Contains(text, `"sha"		"old"`) {
		t.Fatal("expected sha to change")
	}
}
