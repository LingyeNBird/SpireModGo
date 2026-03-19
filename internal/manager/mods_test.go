package manager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetInstallNamePrefersManifest(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	modDir := filepath.Join(root, "DamageMeter_v1.7.6")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "mod_manifest.json"), []byte(`{"pck_name":"DamageMeter","name":"Damage Meter","version":"1.7.6"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "OtherName.dll"), []byte("dll"), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest, err := readManifest(modDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := getInstallName(modDir, manifest); got != "DamageMeter" {
		t.Fatalf("expected manifest install name, got %q", got)
	}
}

func TestInstallModsCopiesFilesAndEnablesSettings(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	m, err := New(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	m.SaveRoot = filepath.Join(baseDir, "steam")

	modDir := filepath.Join(m.ModsSource, "SpeedX_v0.8.6")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "mod_manifest.json"), []byte(`{"pck_name":"SpeedX","name":"SpeedX","version":"0.8.6","author":"tester"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "SpeedX.dll"), []byte("dll"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "SpeedX.pck"), []byte("pck"), 0o644); err != nil {
		t.Fatal(err)
	}

	settingsPath := filepath.Join(m.SaveRoot, "76561197960265729", "settings.save")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte(`{"mod_settings":{"mods_enabled":false}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	gameDir := filepath.Join(baseDir, "game")
	if err := os.MkdirAll(gameDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mods, err := m.ListAvailableMods(gameDir)
	if err != nil {
		t.Fatal(err)
	}
	results, err := m.InstallMods(gameDir, mods)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 install result, got %d", len(results))
	}
	if results[0].FilesCopied != 3 {
		t.Fatalf("expected 3 copied files, got %d", results[0].FilesCopied)
	}
	if !fileExists(filepath.Join(gameDir, "mods", "SpeedX", "SpeedX.dll")) {
		t.Fatal("expected dll to be copied")
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == `{"mod_settings":{"mods_enabled":false}}`+"\n" {
		t.Fatal("expected settings.save to be updated")
	}
}

func TestSyncLocalModsPromotesNewerInstalledVersion(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	m, err := New(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	localDir := filepath.Join(m.ModsSource, "SpeedX_v0.8.6")
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localDir, "mod_manifest.json"), []byte(`{"pck_name":"SpeedX","version":"0.8.6"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	gameDir := filepath.Join(baseDir, "game")
	installedDir := filepath.Join(gameDir, "mods", "SpeedX")
	if err := os.MkdirAll(installedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installedDir, "mod_manifest.json"), []byte(`{"pck_name":"SpeedX","version":"0.9.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installedDir, "SpeedX.dll"), []byte("dll"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := m.SyncLocalMods(gameDir); err != nil {
		t.Fatal(err)
	}
	updatedManifest, err := os.ReadFile(filepath.Join(m.ModsSource, "SpeedX_v0.9.0", "mod_manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(updatedManifest) != `{"pck_name":"SpeedX","version":"0.9.0"}` {
		t.Fatalf("unexpected synced manifest: %q", string(updatedManifest))
	}
}

func TestListAvailableModsMarksLegacyManifestForRepair(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	m, err := New(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	gameDir := filepath.Join(baseDir, "game")
	if err := os.MkdirAll(filepath.Join(gameDir, "mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	modDir := filepath.Join(m.ModsSource, "LegacyDamage")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "LegacyDamage.pck"), []byte("pck"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "mod_manifest.json"), []byte(`{"pck_name":"LegacyDamage","name":"LegacyDamage","version":"1.0.0","author":"LegacyDamage"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	mods, err := m.ListAvailableMods(gameDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || !mods[0].NeedsRepair {
		t.Fatalf("expected available mod to require repair, got %+v", mods)
	}
}

func TestRepairModCreatesTargetJsonAndRemovesLegacyManifest(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	m, err := New(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	modDir := filepath.Join(baseDir, "LegacyDamage")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "DamageMeter.pck"), []byte("pck"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "mod_manifest.json"), []byte(`{"version":"1.2.3"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := m.RepairMod(modDir)
	if err != nil {
		t.Fatalf("repair mod: %v", err)
	}
	if filepath.Base(result.ConfigPath) != "DamageMeter.json" {
		t.Fatalf("expected repaired config path to use pck basename, got %q", result.ConfigPath)
	}
	if !result.RemovedLegacyManifest {
		t.Fatal("expected legacy manifest to be removed")
	}
	if fileExists(filepath.Join(modDir, "mod_manifest.json")) {
		t.Fatal("expected mod_manifest.json to be removed")
	}
	data, err := os.ReadFile(filepath.Join(modDir, "DamageMeter.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["id"] != "DamageMeter" || manifest["pck_name"] != "DamageMeter" || manifest["author"] != "DamageMeter" {
		t.Fatalf("expected repaired config to normalize names, got %v", manifest)
	}
	if manifest["has_pck"] != true {
		t.Fatalf("expected repaired config to set has_pck, got %v", manifest["has_pck"])
	}
}
