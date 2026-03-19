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
var loginUserBlockRE = regexp.MustCompile(`(?s)"(\d+)"\s*\{(.*?)\}`)
var personaNameRE = regexp.MustCompile(`"PersonaName"\s+"([^"]*)"`)
var accountNameRE = regexp.MustCompile(`"AccountName"\s+"([^"]*)"`)

type SteamProfile struct {
	SteamID     string
	DisplayName string
}

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

func (m *Manager) ListSteamProfiles() ([]SteamProfile, error) {
	ids, err := m.ListSteamIDs()
	if err != nil {
		return nil, err
	}
	names := m.loadSteamDisplayNames()
	profiles := make([]SteamProfile, 0, len(ids))
	for _, steamID := range ids {
		displayName := names[steamID]
		if strings.TrimSpace(displayName) == "" {
			displayName = steamID
		}
		profiles = append(profiles, SteamProfile{SteamID: steamID, DisplayName: displayName})
	}
	return profiles, nil
}

func (m *Manager) loadSteamDisplayNames() map[string]string {
	steamPath := m.getSteamPath()
	if strings.TrimSpace(steamPath) == "" {
		return map[string]string{}
	}
	path := filepath.Join(steamPath, "config", "loginusers.vdf")
	content, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}
	names, err := parseLoginUsersDisplayNames(content)
	if err != nil {
		m.logf("parse loginusers.vdf failed: %v", err)
		return map[string]string{}
	}
	return names
}

func parseLoginUsersDisplayNames(content []byte) (map[string]string, error) {
	text := string(content)
	matches := loginUserBlockRE.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil, errors.New("no Steam users found in loginusers.vdf")
	}
	profiles := make(map[string]string, len(matches))
	for _, match := range matches {
		steamID := match[1]
		block := match[2]
		displayName := firstSubmatch(personaNameRE, block)
		if strings.TrimSpace(displayName) == "" {
			displayName = firstSubmatch(accountNameRE, block)
		}
		profiles[steamID] = strings.TrimSpace(displayName)
	}
	return profiles, nil
}

func firstSubmatch(re *regexp.Regexp, text string) string {
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}
