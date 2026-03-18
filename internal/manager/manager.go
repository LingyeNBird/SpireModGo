package manager

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Manager struct {
	BaseDir           string
	ConfigPath        string
	ModsSource        string
	SaveRoot          string
	LogDir            string
	Config            Config
	logger            *log.Logger
	logFile           *os.File
	steamPathOverride string
}

func New(baseDir string) (*Manager, error) {
	if baseDir == "" {
		resolved, err := resolveBaseDir()
		if err != nil {
			return nil, err
		}
		baseDir = resolved
	}

	baseDir = filepath.Clean(baseDir)
	m := &Manager{
		BaseDir:    baseDir,
		ConfigPath: filepath.Join(baseDir, "modmanager.json"),
		ModsSource: filepath.Join(baseDir, "Mods"),
		SaveRoot:   filepath.Join(os.Getenv("APPDATA"), "SlayTheSpire2", "steam"),
		LogDir:     filepath.Join(baseDir, "logs"),
		Config:     Config{},
	}
	if err := m.initLogger(); err != nil {
		return nil, err
	}
	if err := m.LoadConfig(); err != nil {
		m.logf("load config failed: %v", err)
	}
	return m, nil
}

func resolveBaseDir() (string, error) {
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		if looksLikeAppBase(exeDir) {
			return exeDir, nil
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if looksLikeAppBase(cwd) {
		return cwd, nil
	}
	return cwd, nil
}

func looksLikeAppBase(dir string) bool {
	if dir == "" {
		return false
	}
	for _, candidate := range []string{"Mods", "modmanager.json"} {
		if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
			return true
		}
	}
	return false
}

func (m *Manager) Close() error {
	if m.logFile == nil {
		return nil
	}
	err := m.logFile.Close()
	m.logFile = nil
	return err
}

func (m *Manager) initLogger() error {
	if err := os.MkdirAll(m.LogDir, 0o755); err != nil {
		return err
	}
	if err := m.cleanupOldLogs(); err != nil {
		return err
	}
	logPath := filepath.Join(m.LogDir, fmt.Sprintf("modmanager_%s.log", time.Now().Format("20060102_150405")))
	file, err := os.Create(logPath)
	if err != nil {
		return err
	}
	m.logFile = file
	m.logger = log.New(file, "", log.LstdFlags)
	m.logf("manager started in %s", m.BaseDir)
	return nil
}

func (m *Manager) cleanupOldLogs() error {
	entries, err := os.ReadDir(m.LogDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "modmanager_") && strings.HasSuffix(name, ".log") {
			names = append(names, name)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	for idx, name := range names {
		if idx < 2 {
			continue
		}
		_ = os.Remove(filepath.Join(m.LogDir, name))
	}
	return nil
}

func (m *Manager) logf(format string, args ...any) {
	if m.logger != nil {
		m.logger.Printf(format, args...)
	}
}
