package manager

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func (m *Manager) EnableModsInSettings() (bool, error) {
	entries, err := os.ReadDir(m.SaveRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	changed := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		settingsPath := filepath.Join(m.SaveRoot, entry.Name(), "settings.save")
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			continue
		}
		data = stripUTF8BOM(data)
		var payload map[string]any
		if err := json.Unmarshal(data, &payload); err != nil {
			continue
		}
		modSettings, ok := payload["mod_settings"].(map[string]any)
		if !ok {
			continue
		}
		if enabled, ok := modSettings["mods_enabled"].(bool); ok && enabled {
			continue
		}
		modSettings["mods_enabled"] = true
		payload["mod_settings"] = modSettings
		encoded, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return changed, err
		}
		if err := os.WriteFile(settingsPath, append(encoded, '\n'), 0o644); err != nil {
			return changed, err
		}
		changed = true
	}
	return changed, nil
}
