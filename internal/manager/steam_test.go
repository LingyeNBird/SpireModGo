package manager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindGameDirUsesLibraryFolders(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	m, err := New(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	steamPath := filepath.Join(baseDir, "Steam")
	libPath := filepath.Join(baseDir, "SteamLibrary With Space")
	if err := os.MkdirAll(filepath.Join(steamPath, "steamapps"), 0o755); err != nil {
		t.Fatal(err)
	}
	vdf := `"libraryfolders" { "0" { "path" "` + filepath.ToSlash(libPath) + `" } }`
	if err := os.WriteFile(filepath.Join(steamPath, "steamapps", "libraryfolders.vdf"), []byte(vdf), 0o644); err != nil {
		t.Fatal(err)
	}
	gameDir := filepath.Join(libPath, "steamapps", "common", sts2DirName)
	if err := os.MkdirAll(gameDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gameDir, sts2ExeName), []byte("exe"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.steamPathOverride = steamPath

	found, err := m.FindGameDir()
	if err != nil {
		t.Fatal(err)
	}
	if found != gameDir {
		t.Fatalf("expected %q, got %q", gameDir, found)
	}
}
