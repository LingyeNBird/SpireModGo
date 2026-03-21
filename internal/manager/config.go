package manager

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func (m *Manager) LoadConfig() error {
	data, err := os.ReadFile(m.ConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			legacyPath := m.legacyConfigPath()
			if legacyPath != "" {
				legacyData, legacyErr := os.ReadFile(legacyPath)
				if legacyErr == nil {
					data = legacyData
					err = nil
					if saveErr := os.MkdirAll(filepath.Dir(m.ConfigPath), 0o755); saveErr != nil {
						return saveErr
					}
					if saveErr := os.WriteFile(m.ConfigPath, legacyData, 0o644); saveErr != nil {
						return saveErr
					}
					m.logf("migrated legacy config from %s to %s", legacyPath, m.ConfigPath)
				} else if !errors.Is(legacyErr, os.ErrNotExist) {
					return legacyErr
				}
			}
			if len(data) == 0 {
				m.Config = Config{}
				return nil
			}
		}
		if err != nil {
			return err
		}
	}
	data = stripUTF8BOM(data)
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		m.Config = Config{}
		return err
	}
	m.Config = cfg
	return nil
}

func (m *Manager) SaveConfig() error {
	data, err := json.MarshalIndent(m.Config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.ConfigPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(m.ConfigPath, append(data, '\n'), 0o644)
}

func (m *Manager) GetGameDir() string {
	dir := stringsTrimSpace(m.Config.GameDir)
	if dir == "" {
		return ""
	}
	if fileExists(filepath.Join(dir, sts2ExeName)) {
		return dir
	}
	m.Config.GameDir = ""
	_ = m.SaveConfig()
	return ""
}

func (m *Manager) SetGameDir(dir string) error {
	dir = stringsTrimSpace(dir)
	if dir == "" {
		return errors.New("game directory is empty")
	}
	if !fileExists(filepath.Join(dir, sts2ExeName)) {
		return errors.New("SlayTheSpire2.exe not found in selected directory")
	}
	m.Config.GameDir = filepath.Clean(dir)
	return m.SaveConfig()
}

func (m *Manager) ClearConfig() error {
	m.Config = Config{}
	if err := os.Remove(m.ConfigPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func stringsTrimSpace(value string) string {
	for len(value) > 0 && (value[0] == ' ' || value[0] == '\t' || value[0] == '"') {
		value = value[1:]
	}
	for len(value) > 0 && (value[len(value)-1] == ' ' || value[len(value)-1] == '\t' || value[len(value)-1] == '"') {
		value = value[:len(value)-1]
	}
	return value
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
