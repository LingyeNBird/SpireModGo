package manager

import (
	"archive/zip"
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

func TestInspectModRepairNeedDoesNotFlagTargetJSONWithCustomMetadata(t *testing.T) {
	t.Parallel()
	modDir := filepath.Join(t.TempDir(), "CustomDamage")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "DamageMeter.pck"), []byte("pck"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "DamageMeter.dll"), []byte("dll"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "DamageMeter.json"), []byte(`{"id":"custom-id","name":"Fancy Name","version":"1.0.0","author":"Alice","pck_name":"FancyInstall","has_pck":false,"has_dll":false}`), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest, err := readManifest(modDir)
	if err != nil {
		t.Fatal(err)
	}
	needsRepair, hint := inspectModRepairNeed(modDir, manifest)
	if needsRepair {
		t.Fatalf("expected target json with custom metadata to skip repair, got hint %q", hint)
	}
}

func TestInspectModRepairNeedDoesNotFlagTargetAndLegacyCoexistence(t *testing.T) {
	t.Parallel()
	modDir := filepath.Join(t.TempDir(), "LegacyAndTarget")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "DamageMeter.pck"), []byte("pck"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "DamageMeter.json"), []byte(`{"pck_name":"DamageMeter","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "mod_manifest.json"), []byte(`{"pck_name":"LegacyDamage","version":"0.9.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest, err := readManifest(modDir)
	if err != nil {
		t.Fatal(err)
	}
	needsRepair, hint := inspectModRepairNeed(modDir, manifest)
	if needsRepair {
		t.Fatalf("expected target+legacy coexistence to skip repair, got hint %q", hint)
	}
}

func TestInspectModRepairNeedFlagsLegacyOnly(t *testing.T) {
	t.Parallel()
	modDir := filepath.Join(t.TempDir(), "LegacyOnly")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "DamageMeter.pck"), []byte("pck"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "mod_manifest.json"), []byte(`{"pck_name":"DamageMeter","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest, err := readManifest(modDir)
	if err != nil {
		t.Fatal(err)
	}
	needsRepair, hint := inspectModRepairNeed(modDir, manifest)
	if !needsRepair || hint != "missing_target_json" {
		t.Fatalf("expected legacy-only manifest to require target json repair, got needsRepair=%v hint=%q", needsRepair, hint)
	}
}

func TestInspectModRepairNeedFlagsMissingManifest(t *testing.T) {
	t.Parallel()
	modDir := filepath.Join(t.TempDir(), "MissingManifest")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "DamageMeter.pck"), []byte("pck"), 0o644); err != nil {
		t.Fatal(err)
	}

	needsRepair, hint := inspectModRepairNeed(modDir, nil)
	if !needsRepair || hint != "missing_target_json" {
		t.Fatalf("expected missing manifest to require target json repair, got needsRepair=%v hint=%q", needsRepair, hint)
	}
}

func TestRepairModCreatesTargetJsonAndKeepsLegacyManifest(t *testing.T) {
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
	if result.RemovedLegacyManifest {
		t.Fatal("expected repair to keep legacy manifest for script parity")
	}
	if !fileExists(filepath.Join(modDir, "mod_manifest.json")) {
		t.Fatal("expected mod_manifest.json to remain after repair")
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

func TestRepairModPreservesExistingMetadataAndFillsMissingFields(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	m, err := New(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	modDir := filepath.Join(baseDir, "LegacyCustom")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "DamageMeter.pck"), []byte("pck"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "DamageMeter.dll"), []byte("dll"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "mod_manifest.json"), []byte(`{"id":"custom-id","name":"Fancy Name","version":"1.2.3","author":"Alice","pck_name":"FancyInstall","has_pck":false,"has_dll":false}`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := m.RepairMod(modDir)
	if err != nil {
		t.Fatalf("repair mod: %v", err)
	}
	if filepath.Base(result.ConfigPath) != "DamageMeter.json" {
		t.Fatalf("expected repaired config path to use pck basename, got %q", result.ConfigPath)
	}
	data, err := os.ReadFile(result.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["id"] != "custom-id" || manifest["name"] != "Fancy Name" || manifest["author"] != "Alice" || manifest["pck_name"] != "FancyInstall" {
		t.Fatalf("expected repair to preserve existing metadata, got %v", manifest)
	}
	if manifest["has_pck"] != false || manifest["has_dll"] != false {
		t.Fatalf("expected repair to preserve existing asset flags, got has_pck=%v has_dll=%v", manifest["has_pck"], manifest["has_dll"])
	}
	if manifest["description"] != "这是通过脚本自动生成的临时配置文件，如果mod更新请使用mod作者提供的新文件" {
		t.Fatalf("expected repair to fill missing description, got %v", manifest["description"])
	}
	if manifest["affects_gameplay"] != false {
		t.Fatalf("expected repair to fill missing affects_gameplay, got %v", manifest["affects_gameplay"])
	}
	deps, ok := manifest["dependencies"].([]any)
	if !ok || len(deps) != 0 {
		t.Fatalf("expected repair to fill missing dependencies with empty array, got %T %v", manifest["dependencies"], manifest["dependencies"])
	}
}

func TestImportAvailableModsFromZipFallsBackToUserModsRoot(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	m, err := New(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	m.UserModsSource = filepath.Join(baseDir, "userdata", "SpireModGo", "mods")

	zipPath := filepath.Join(baseDir, "SpeedX.zip")
	writeTestZip(t, zipPath, map[string]string{
		"SpeedX.dll":               "dll",
		"SpeedX.pck":               "pck",
		"SpeedX.json":              `{"pck_name":"SpeedX","name":"SpeedX","version":"1.0.0"}`,
		"data/config/settings.txt": "nested",
	})

	results, err := m.ImportAvailableModsFromZip(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one imported mod, got %d", len(results))
	}
	if results[0].Destination != filepath.Join(m.UserModsSource, "SpeedX") {
		t.Fatalf("unexpected destination: %s", results[0].Destination)
	}
	if !fileExists(filepath.Join(m.UserModsSource, "SpeedX", "SpeedX.dll")) {
		t.Fatal("expected dll to be imported into user mods root")
	}
	if !fileExists(filepath.Join(m.UserModsSource, "SpeedX", "data", "config", "settings.txt")) {
		t.Fatal("expected nested files to be imported")
	}
}

func TestImportInstalledModsFromZipEnablesSettings(t *testing.T) {
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
	if err := os.WriteFile(settingsPath, []byte(`{"mod_settings":{"mods_enabled":false}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(baseDir, "bundle.zip")
	writeTestZip(t, zipPath, map[string]string{
		"folder/SpeedX.dll":  "dll",
		"folder/SpeedX.pck":  "pck",
		"folder/SpeedX.json": `{"pck_name":"SpeedX","name":"SpeedX","version":"1.0.0"}`,
	})

	gameDir := filepath.Join(baseDir, "game")
	results, err := m.ImportInstalledModsFromZip(gameDir, zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].EnableChanged {
		t.Fatalf("expected one imported mod with enable change, got %+v", results)
	}
	if !fileExists(filepath.Join(gameDir, "mods", "SpeedX", "SpeedX.dll")) {
		t.Fatal("expected dll to be imported into game mods root")
	}
}

func TestExportAvailableModsToZipWritesSelectedFolders(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	m, err := New(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	modDir := filepath.Join(m.ModsSource, "SpeedX_v1.0.0")
	if err := os.MkdirAll(filepath.Join(modDir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "SpeedX.dll"), []byte("dll"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "data", "settings.txt"), []byte("nested"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := m.ExportAvailableModsToZip([]ModPackage{{DirName: "SpeedX_v1.0.0", SourcePath: modDir}}, filepath.Join(baseDir, "export.zip"))
	if err != nil {
		t.Fatal(err)
	}
	if result.FilesAdded != 2 {
		t.Fatalf("expected two files in export zip, got %d", result.FilesAdded)
	}
	reader, err := zip.OpenReader(result.ZipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	seen := map[string]bool{}
	for _, file := range reader.File {
		seen[file.Name] = true
	}
	if !seen["SpeedX_v1.0.0/SpeedX.dll"] || !seen["SpeedX_v1.0.0/data/settings.txt"] {
		t.Fatalf("unexpected archive contents: %v", seen)
	}
}

func TestPreviewZipModsUsesDLLNameWithoutManifest(t *testing.T) {
	t.Parallel()
	zipPath := filepath.Join(t.TempDir(), "bundle.zip")
	writeTestZip(t, zipPath, map[string]string{
		"folder/SpeedX.dll": "dll",
		"folder/SpeedX.pck": "pck",
	})
	scans, err := scanZipModCandidates(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(scans) != 1 || scans[0].Name != "SpeedX" {
		t.Fatalf("expected DLL-derived mod name, got %+v", scans)
	}
}

func TestPreviewZipModsIgnoresUnrelatedJSON(t *testing.T) {
	t.Parallel()
	zipPath := filepath.Join(t.TempDir(), "bundle.zip")
	writeTestZip(t, zipPath, map[string]string{
		"folder/SpeedX.dll":  "dll",
		"folder/SpeedX.pck":  "pck",
		"folder/data.json":   `{"name":"wrong"}`,
		"folder/SpeedX.json": `{"pck_name":"SpeedX","name":"SpeedX"}`,
	})
	scans, err := scanZipModCandidates(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(scans) != 1 || scans[0].Name != "SpeedX" {
		t.Fatalf("expected mod manifest naming, got %+v", scans)
	}
}

func writeTestZip(t *testing.T, zipPath string, files map[string]string) {
	t.Helper()
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	writer := zip.NewWriter(file)
	for name, data := range files {
		entry, err := writer.Create(name)
		if err != nil {
			_t := writer.Close()
			_ = _t
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(data)); err != nil {
			_t := writer.Close()
			_ = _t
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
}
