package manager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type UninstallResult struct {
	Name string
	Err  error
}

func (m *Manager) ListAvailableMods(gameDir string) ([]ModPackage, error) {
	entries, err := os.ReadDir(m.ModsSource)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	installed, err := m.ListInstalledMods(gameDir)
	if err != nil {
		return nil, err
	}
	installedMap := map[string]InstalledMod{}
	for _, mod := range installed {
		installedMap[mod.DirName] = mod
	}

	mods := make([]ModPackage, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		fullPath := filepath.Join(m.ModsSource, entry.Name())
		manifest, _ := readManifest(fullPath)
		installName := getInstallName(fullPath, manifest)
		label := formatModLabel(entry.Name(), manifest, installName)
		mod := ModPackage{
			DirName:     entry.Name(),
			SourcePath:  fullPath,
			InstallName: installName,
			Label:       label,
			Manifest:    manifest,
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
		mods = append(mods, InstalledMod{
			DirName:  entry.Name(),
			FullPath: fullPath,
			Manifest: manifest,
			Label:    formatModLabel(entry.Name(), manifest, getInstallName(fullPath, manifest)),
		})
	}
	sort.Slice(mods, func(i, j int) bool {
		return strings.ToLower(mods[i].Label) < strings.ToLower(mods[j].Label)
	})
	return mods, nil
}

func readManifest(modDir string) (*ModManifest, error) {
	manifestPath := filepath.Join(modDir, "mod_manifest.json")
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

func getInstallName(modDir string, manifest *ModManifest) string {
	if manifest != nil && stringsTrimSpace(manifest.PckName) != "" {
		return manifest.PckName
	}
	entries, _ := os.ReadDir(modDir)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.Contains(name, ".bak") || filepath.Ext(name) != ".dll" {
			continue
		}
		return strings.TrimSuffix(name, filepath.Ext(name))
	}
	return filepath.Base(modDir)
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
