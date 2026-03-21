package manager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSpireModGoUserDataRootPrefersUserDataParent(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join(`C:\workspace`, `SpireModGo`)
	userDataParent := filepath.Join(`C:\Users`, `tester`, `AppData`, `Roaming`)
	got := spireModGoUserDataRoot(baseDir, userDataParent)
	want := filepath.Join(userDataParent, "SpireModGo")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSpireModGoUserDataRootFallsBackToBaseDir(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	got := spireModGoUserDataRoot(baseDir, "")
	want := filepath.Join(baseDir, "SpireModGo")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestLooksLikeAppBaseRequiresModsDirectory(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(baseDir, "modmanager.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if looksLikeAppBase(baseDir) {
		t.Fatal("expected modmanager.json alone to no longer qualify as app base")
	}
	modsDir := filepath.Join(baseDir, "Mods")
	if err := os.MkdirAll(modsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if !looksLikeAppBase(baseDir) {
		t.Fatal("expected Mods directory to qualify as app base")
	}
}

func TestLoadConfigMigratesLegacyBaseDirConfig(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	legacyConfigPath := filepath.Join(baseDir, "modmanager.json")
	newConfigPath := filepath.Join(baseDir, "userdata", "SpireModGo", "modmanager.json")
	legacyContent := []byte(`{"GameDir":"C:\\Games\\SlayTheSpire2"}`)
	if err := os.WriteFile(legacyConfigPath, legacyContent, 0o644); err != nil {
		t.Fatal(err)
	}
	m := &Manager{BaseDir: baseDir, ConfigPath: newConfigPath}
	if err := m.LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if m.Config.GameDir != `C:\Games\SlayTheSpire2` {
		t.Fatalf("expected migrated GameDir to be loaded, got %q", m.Config.GameDir)
	}
	migratedContent, err := os.ReadFile(newConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var migrated Config
	if err := json.Unmarshal(migratedContent, &migrated); err != nil {
		t.Fatal(err)
	}
	if migrated.GameDir != m.Config.GameDir {
		t.Fatalf("expected migrated config to match loaded config, got %q", migrated.GameDir)
	}
}

func TestInitLoggerUsesCollisionSafeLogNames(t *testing.T) {
	t.Parallel()
	logDir := filepath.Join(t.TempDir(), "logs")
	first := &Manager{LogDir: logDir}
	if err := first.initLogger(); err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second := &Manager{LogDir: logDir}
	if err := second.initLogger(); err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if first.logFile == nil || second.logFile == nil {
		t.Fatal("expected both logger instances to create log files")
	}
	if filepath.Base(first.logFile.Name()) == filepath.Base(second.logFile.Name()) {
		t.Fatalf("expected unique log file names, got %q", filepath.Base(first.logFile.Name()))
	}
}
