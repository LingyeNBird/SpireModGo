package manager

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var libraryPathRE = regexp.MustCompile(`"path"\s+"([^"]+)"`)

func (m *Manager) EnsureGameDir() (string, error) {
	if dir := m.GetGameDir(); dir != "" {
		return dir, nil
	}
	dir, err := m.FindGameDir()
	if err != nil {
		return "", err
	}
	if dir != "" {
		m.Config.GameDir = dir
		if saveErr := m.SaveConfig(); saveErr != nil {
			m.logf("save config failed: %v", saveErr)
		}
	}
	return dir, nil
}

func (m *Manager) FindGameDir() (string, error) {
	steamPath := m.getSteamPath()
	if steamPath != "" {
		for _, candidate := range m.steamCandidates(steamPath) {
			if fileExists(filepath.Join(candidate, sts2ExeName)) {
				return candidate, nil
			}
		}
	}
	for _, root := range existingDriveRoots() {
		for _, sub := range commonSteamSubdirs() {
			candidate := filepath.Join(root, sub)
			if fileExists(filepath.Join(candidate, sts2ExeName)) {
				return candidate, nil
			}
		}
	}
	return "", nil
}

func (m *Manager) getSteamPath() string {
	if m.steamPathOverride != "" {
		return filepath.Clean(m.steamPathOverride)
	}
	return readSteamPathFromRegistry()
}

func (m *Manager) steamCandidates(steamPath string) []string {
	var candidates []string
	steamPath = filepath.Clean(strings.ReplaceAll(steamPath, "/", `\`))
	candidates = append(candidates, filepath.Join(steamPath, "steamapps", "common", sts2DirName))
	for _, vdfPath := range []string{
		filepath.Join(steamPath, "steamapps", "libraryfolders.vdf"),
		filepath.Join(steamPath, "config", "libraryfolders.vdf"),
	} {
		content, err := os.ReadFile(vdfPath)
		if err != nil {
			continue
		}
		matches := libraryPathRE.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			libPath := strings.ReplaceAll(match[1], `\\`, `\`)
			candidates = append(candidates, filepath.Join(libPath, "steamapps", "common", sts2DirName))
		}
		break
	}
	return uniqueStrings(candidates)
}

func existingDriveRoots() []string {
	var roots []string
	for drive := 'C'; drive <= 'Z'; drive++ {
		root := string([]rune{drive}) + `:\`
		if info, err := os.Stat(root); err == nil && info.IsDir() {
			roots = append(roots, root)
		}
	}
	return roots
}

func commonSteamSubdirs() []string {
	return []string{
		filepath.Join("SteamLibrary", "steamapps", "common", sts2DirName),
		filepath.Join("Steam", "steamapps", "common", sts2DirName),
		filepath.Join("Program Files (x86)", "Steam", "steamapps", "common", sts2DirName),
		filepath.Join("Program Files", "Steam", "steamapps", "common", sts2DirName),
		filepath.Join("Games", "Steam", "steamapps", "common", sts2DirName),
		filepath.Join("Games", "SteamLibrary", "steamapps", "common", sts2DirName),
		filepath.Join("Game", "Steam", "steamapps", "common", sts2DirName),
		filepath.Join("Game", "SteamLibrary", "steamapps", "common", sts2DirName),
	}
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = filepath.Clean(value)
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (m *Manager) ListSteamIDs() ([]string, error) {
	entries, err := os.ReadDir(m.SaveRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			ids = append(ids, entry.Name())
		}
	}
	sort.Strings(ids)
	return ids, nil
}
