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
	UserModsSource    string
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
	userDataRoot := spireModGoUserDataRoot(baseDir, resolveUserDataParentDir())
	m := &Manager{
		BaseDir:        baseDir,
		ConfigPath:     filepath.Join(userDataRoot, "modmanager.json"),
		ModsSource:     filepath.Join(baseDir, "Mods"),
		UserModsSource: filepath.Join(userDataRoot, "mods"),
		SaveRoot:       filepath.Join(os.Getenv("APPDATA"), "SlayTheSpire2", "steam"),
		LogDir:         filepath.Join(userDataRoot, "logs"),
		Config:         Config{},
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
	for _, candidate := range []string{"Mods"} {
		if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
			return true
		}
	}
	return false
}

func resolveUserDataParentDir() string {
	if dir := stringsTrimSpace(os.Getenv("APPDATA")); dir != "" {
		return dir
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return stringsTrimSpace(dir)
}

func spireModGoUserDataRoot(baseDir, userDataParent string) string {
	if userDataParent != "" {
		return filepath.Join(userDataParent, "SpireModGo")
	}
	if baseDir == "" {
		return "SpireModGo"
	}
	return filepath.Join(baseDir, "SpireModGo")
}

func (m *Manager) legacyConfigPath() string {
	if m.BaseDir == "" {
		return ""
	}
	legacyPath := filepath.Join(m.BaseDir, "modmanager.json")
	if filepath.Clean(legacyPath) == filepath.Clean(m.ConfigPath) {
		return ""
	}
	return legacyPath
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
	logPath := filepath.Join(m.LogDir, fmt.Sprintf("modmanager_%s_%d.log", time.Now().Format("20060102_150405.000000000"), os.Getpid()))
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

func (m *Manager) PreferredAvailableModsRoot() string {
	if dirExists(m.ModsSource) {
		return m.ModsSource
	}
	return m.UserModsSource
}

func (m *Manager) AvailableModsRoots() []string {
	roots := make([]string, 0, 2)
	seen := map[string]bool{}
	for _, root := range []string{m.ModsSource, m.UserModsSource} {
		clean := filepath.Clean(root)
		if seen[clean] {
			continue
		}
		seen[clean] = true
		if dirExists(clean) {
			roots = append(roots, clean)
		}
	}
	if len(roots) == 0 {
		return []string{m.PreferredAvailableModsRoot()}
	}
	return roots
}

func (m *Manager) DisplayAvailableModsRoot() string {
	roots := m.AvailableModsRoots()
	if len(roots) == 0 {
		return m.PreferredAvailableModsRoot()
	}
	return strings.Join(roots, " | ")
}
