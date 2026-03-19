package manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var errRepairAmbiguousLayout = errors.New("mod folder has multiple .pck or .dll files")

type modRepairLayout struct {
	FolderName     string
	ModDir         string
	BaseName       string
	TargetJSONPath string
	LegacyJSONPath string
	HasPck         bool
	HasDll         bool
	PckCount       int
	DllCount       int
	TargetExists   bool
	LegacyExists   bool
	ConfigPath     string
	Config         *ModManifest
	ConfigParseErr error
}

type UninstallResult struct {
	Name string
	Err  error
}

func (m *Manager) ListAvailableMods(gameDir string) ([]ModPackage, error) {
	installed, err := m.ListInstalledMods(gameDir)
	if err != nil {
		return nil, err
	}
	installedMap := map[string]InstalledMod{}
	for _, mod := range installed {
		installedMap[mod.DirName] = mod
	}

	mods := make([]ModPackage, 0)
	for _, root := range m.AvailableModsRoots() {
		entries, err := os.ReadDir(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			fullPath := filepath.Join(root, entry.Name())
			manifest, _ := readManifest(fullPath)
			installName := getInstallName(fullPath, manifest)
			needsRepair, repairHint := inspectModRepairNeed(fullPath, manifest)
			label := formatModLabel(entry.Name(), manifest, installName)
			mod := ModPackage{
				DirName:     entry.Name(),
				SourcePath:  fullPath,
				InstallName: installName,
				Label:       label,
				Manifest:    manifest,
				NeedsRepair: needsRepair,
				RepairHint:  repairHint,
			}
			if installedMod, ok := installedMap[installName]; ok {
				mod.Installed = true
				if installedMod.Manifest != nil {
					mod.InstalledVersion = installedMod.Manifest.Version
				}
				if manifest != nil && installedMod.Manifest != nil && manifest.Version != "" && installedMod.Manifest.Version != "" && manifest.Version != installedMod.Manifest.Version {
					mod.Updatable = true
				}
			}
			mods = append(mods, mod)
		}
	}
	sort.Slice(mods, func(i, j int) bool {
		return strings.ToLower(mods[i].Label) < strings.ToLower(mods[j].Label)
	})
	return mods, nil
}

func (m *Manager) ListInstalledMods(gameDir string) ([]InstalledMod, error) {
	modsDir := filepath.Join(gameDir, "mods")
	entries, err := os.ReadDir(modsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	mods := make([]InstalledMod, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		fullPath := filepath.Join(modsDir, entry.Name())
		manifest, _ := readManifest(fullPath)
		needsRepair, repairHint := inspectModRepairNeed(fullPath, manifest)
		mods = append(mods, InstalledMod{
			DirName:     entry.Name(),
			FullPath:    fullPath,
			Manifest:    manifest,
			Label:       formatModLabel(entry.Name(), manifest, getInstallName(fullPath, manifest)),
			NeedsRepair: needsRepair,
			RepairHint:  repairHint,
		})
	}
	sort.Slice(mods, func(i, j int) bool {
		return strings.ToLower(mods[i].Label) < strings.ToLower(mods[j].Label)
	})
	return mods, nil
}

func readManifest(modDir string) (*ModManifest, error) {
	manifestPath := findManifestPath(modDir)
	if manifestPath == "" {
		return nil, os.ErrNotExist
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	data = stripUTF8BOM(data)
	var manifest ModManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func findManifestPath(modDir string) string {
	layout := inspectModRepairLayout(modDir)
	if layout.TargetExists {
		return layout.TargetJSONPath
	}
	if layout.LegacyExists {
		return layout.LegacyJSONPath
	}
	return ""
}

func getInstallName(modDir string, manifest *ModManifest) string {
	entries, _ := os.ReadDir(modDir)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	return installNameFromFileNames(filepath.Base(modDir), names, manifest)
}

func installNameFromFileNames(dirName string, fileNames []string, manifest *ModManifest) string {
	if manifest != nil && stringsTrimSpace(manifest.PckName) != "" {
		return manifest.PckName
	}
	for _, name := range fileNames {
		if strings.Contains(name, ".bak") || filepath.Ext(name) != ".dll" {
			continue
		}
		return strings.TrimSuffix(name, filepath.Ext(name))
	}
	return dirName
}

func formatModLabel(dirName string, manifest *ModManifest, installName string) string {
	if manifest == nil {
		if installName != dirName {
			return fmt.Sprintf("%s (%s) v?", installName, dirName)
		}
		return dirName + " v?"
	}
	label := dirName
	if stringsTrimSpace(manifest.Name) != "" && manifest.Name != dirName {
		label = fmt.Sprintf("%s (%s)", manifest.Name, dirName)
	}
	version := manifest.Version
	if version == "" {
		version = "unknown"
	}
	label += " v" + version
	if stringsTrimSpace(manifest.Author) != "" {
		label += " by " + manifest.Author
	}
	return label
}

func (m *Manager) InstallMods(gameDir string, mods []ModPackage) ([]InstallResult, error) {
	destRoot := filepath.Join(gameDir, "mods")
	if err := ensureDir(destRoot); err != nil {
		return nil, err
	}

	results := make([]InstallResult, 0, len(mods))
	for _, mod := range mods {
		result := InstallResult{Mod: mod}
		destDir := filepath.Join(destRoot, mod.InstallName)
		if err := ensureDir(destDir); err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(mod.SourcePath)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			srcPath := filepath.Join(mod.SourcePath, entry.Name())
			dstPath := filepath.Join(destDir, entry.Name())
			replaced, backupName, err := copyFileWithReplaceFallback(srcPath, dstPath)
			fileResult := InstallFileResult{Name: entry.Name(), Replaced: replaced, BackupName: backupName, Err: err}
			if err == nil {
				result.FilesCopied++
			}
			result.Files = append(result.Files, fileResult)
		}
		changed, err := m.EnableModsInSettings()
		if err == nil {
			result.EnableChanged = changed
		}
		results = append(results, result)
	}
	return results, nil
}

func (m *Manager) RepairMod(modDir string) (ModRepairResult, error) {
	layout := inspectModRepairLayout(modDir)
	if layout.PckCount > 1 || layout.DllCount > 1 {
		return ModRepairResult{}, errRepairAmbiguousLayout
	}
	defaults := map[string]any{
		"id":               layout.BaseName,
		"name":             layout.BaseName,
		"affects_gameplay": false,
		"has_pck":          layout.HasPck,
		"version":          "0.0.1",
		"description":      "这是通过脚本自动生成的临时配置文件，如果mod更新请使用mod作者提供的新文件",
		"author":           layout.BaseName,
		"pck_name":         layout.BaseName,
		"has_dll":          layout.HasDll,
		"dependencies":     []any{},
	}
	config := map[string]any{}
	if layout.ConfigPath != "" {
		data, err := os.ReadFile(layout.ConfigPath)
		if err != nil {
			return ModRepairResult{}, err
		}
		if err := json.Unmarshal(stripUTF8BOM(data), &config); err != nil {
			return ModRepairResult{}, err
		}
	}
	for key, value := range defaults {
		if _, ok := config[key]; !ok {
			config[key] = value
		}
	}
	config["id"] = layout.BaseName
	config["name"] = layout.BaseName
	config["author"] = layout.BaseName
	config["pck_name"] = layout.BaseName
	config["has_pck"] = layout.HasPck
	config["has_dll"] = layout.HasDll
	if _, ok := config["dependencies"]; !ok {
		config["dependencies"] = []any{}
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return ModRepairResult{}, err
	}
	if err := os.WriteFile(layout.TargetJSONPath, data, 0o644); err != nil {
		return ModRepairResult{}, err
	}
	removedLegacy := false
	if layout.LegacyExists && !sameFilePath(layout.LegacyJSONPath, layout.TargetJSONPath) {
		if err := os.Remove(layout.LegacyJSONPath); err == nil || os.IsNotExist(err) {
			removedLegacy = true
		}
	}
	return ModRepairResult{ConfigPath: layout.TargetJSONPath, RemovedLegacyManifest: removedLegacy}, nil
}

func (m *Manager) UninstallMods(gameDir string, names []string) ([]UninstallResult, error) {
	modsDir := filepath.Join(gameDir, "mods")
	results := make([]UninstallResult, 0, len(names))
	for _, name := range names {
		path := filepath.Join(modsDir, name)
		err := removePathWithRetry(path)
		results = append(results, UninstallResult{Name: name, Err: err})
	}
	return results, nil
}

func (m *Manager) UninstallAllMods(gameDir string) error {
	modsDir := filepath.Join(gameDir, "mods")
	if !dirExists(modsDir) {
		return nil
	}
	entries, err := os.ReadDir(modsDir)
	if err != nil {
		return err
	}
	var firstErr error
	for _, entry := range entries {
		if err := removePathWithRetry(filepath.Join(modsDir, entry.Name())); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *Manager) CleanupBakFiles(gameDir string) ([]string, error) {
	modsDir := filepath.Join(gameDir, "mods")
	var removed []string
	err := filepath.WalkDir(modsDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.Contains(strings.ToLower(d.Name()), ".bak") {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		removed = append(removed, path)
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	return removed, err
}

func (m *Manager) ShouldWarnDisableMods(gameDir string, uninstallAll bool) (bool, error) {
	if uninstallAll {
		return true, nil
	}
	mods, err := m.ListInstalledMods(gameDir)
	if err != nil {
		return false, err
	}
	return len(mods) == 0, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (m *Manager) SyncLocalMods(gameDir string) error {
	installedMods, err := m.ListInstalledMods(gameDir)
	if err != nil {
		return err
	}
	availableMods, err := m.ListAvailableMods(gameDir)
	if err != nil {
		return err
	}
	localByName := map[string]ModPackage{}
	for _, mod := range availableMods {
		localByName[mod.InstallName] = mod
	}
	for _, installed := range installedMods {
		if installed.Manifest == nil || installed.Manifest.PckName == "" || installed.Manifest.Version == "" {
			continue
		}
		local, ok := localByName[installed.Manifest.PckName]
		if !ok || local.Manifest == nil || local.Manifest.Version == "" {
			continue
		}
		if compareVersionStrings(installed.Manifest.Version, local.Manifest.Version) <= 0 {
			continue
		}
		newDir := filepath.Join(m.ModsSource, fmt.Sprintf("%s_v%s", installed.Manifest.PckName, installed.Manifest.Version))
		_ = os.RemoveAll(local.SourcePath)
		if err := ensureDir(newDir); err != nil {
			return err
		}
		entries, err := os.ReadDir(installed.FullPath)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			src := filepath.Join(installed.FullPath, entry.Name())
			dst := filepath.Join(newDir, entry.Name())
			if entry.IsDir() {
				if _, err := copyDirRecursive(src, dst); err != nil {
					return err
				}
				continue
			}
			if err := copyRegularFile(src, dst); err != nil {
				return err
			}
		}
		m.logf("synced local package %s from installed version %s", installed.Manifest.PckName, installed.Manifest.Version)
	}
	return nil
}

func compareVersionStrings(left, right string) int {
	left = strings.TrimPrefix(strings.TrimSpace(left), "v")
	right = strings.TrimPrefix(strings.TrimSpace(right), "v")
	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	maxLen := len(leftParts)
	if len(rightParts) > maxLen {
		maxLen = len(rightParts)
	}
	for idx := 0; idx < maxLen; idx++ {
		li := 0
		ri := 0
		if idx < len(leftParts) {
			li, _ = strconv.Atoi(leftParts[idx])
		}
		if idx < len(rightParts) {
			ri, _ = strconv.Atoi(rightParts[idx])
		}
		if li > ri {
			return 1
		}
		if li < ri {
			return -1
		}
	}
	return strings.Compare(left, right)
}

func inspectModRepairNeed(modDir string, manifest *ModManifest) (bool, string) {
	layout := inspectModRepairLayout(modDir)
	if layout.PckCount > 1 || layout.DllCount > 1 {
		return false, ""
	}
	if layout.LegacyExists && !sameFilePath(layout.LegacyJSONPath, layout.TargetJSONPath) {
		return true, "legacy_manifest"
	}
	if !layout.TargetExists {
		return true, "missing_target_json"
	}
	if manifest == nil {
		return false, ""
	}
	if stringsTrimSpace(manifest.ID) != layout.BaseName || stringsTrimSpace(manifest.Name) != layout.BaseName || stringsTrimSpace(manifest.Author) != layout.BaseName || stringsTrimSpace(manifest.PckName) != layout.BaseName {
		return true, "metadata_mismatch"
	}
	if manifest.HasPck != layout.HasPck || manifest.HasDll != layout.HasDll {
		return true, "asset_flag_mismatch"
	}
	if manifest.Dependencies == nil {
		return true, "missing_dependencies"
	}
	return false, ""
}

func inspectModRepairLayout(modDir string) modRepairLayout {
	folderName := filepath.Base(modDir)
	entries, _ := os.ReadDir(modDir)
	pckNames := make([]string, 0)
	dllCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".pck":
			pckNames = append(pckNames, entry.Name())
		case ".dll":
			dllCount++
		}
	}
	baseName := folderName
	if len(pckNames) == 1 {
		baseName = strings.TrimSuffix(pckNames[0], filepath.Ext(pckNames[0]))
	}
	target := filepath.Join(modDir, baseName+".json")
	legacy := filepath.Join(modDir, "mod_manifest.json")
	configPath := ""
	switch {
	case fileExists(target):
		configPath = target
	case fileExists(legacy):
		configPath = legacy
	}
	return modRepairLayout{
		FolderName:     folderName,
		ModDir:         modDir,
		BaseName:       baseName,
		TargetJSONPath: target,
		LegacyJSONPath: legacy,
		HasPck:         len(pckNames) == 1,
		HasDll:         dllCount == 1,
		PckCount:       len(pckNames),
		DllCount:       dllCount,
		TargetExists:   fileExists(target),
		LegacyExists:   fileExists(legacy),
		ConfigPath:     configPath,
	}
}

func sameFilePath(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}

func removePathWithRetry(path string) error {
	if err := os.RemoveAll(path); err == nil || os.IsNotExist(err) {
		return nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		_ = os.RemoveAll(filepath.Join(path, entry.Name()))
	}
	if err := os.RemoveAll(path); err == nil || os.IsNotExist(err) {
		return nil
	}
	return err
}
