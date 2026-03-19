package manager

import (
	"os"
	"path/filepath"
	"reflect"
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

func TestParseLoginUsersDisplayNames(t *testing.T) {
	vdf := `"users"
{
	"76561198000000001"
	{
		"AccountName"		"alpha_login"
		"PersonaName"		"Alpha"
	}
	"76561198000000002"
	{
		"AccountName"		"beta_login"
	}
}`
	got, err := parseLoginUsersDisplayNames([]byte(vdf))
	if err != nil {
		t.Fatalf("parse loginusers: %v", err)
	}
	want := map[string]string{
		"76561198000000001": "Alpha",
		"76561198000000002": "beta_login",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestListSteamProfilesOverlaysDisplayNames(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	m, err := New(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	m.SaveRoot = filepath.Join(baseDir, "saves")
	if err := os.MkdirAll(filepath.Join(m.SaveRoot, "76561198000000002"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(m.SaveRoot, "76561198000000001"), 0o755); err != nil {
		t.Fatal(err)
	}
	steamPath := filepath.Join(baseDir, "Steam")
	if err := os.MkdirAll(filepath.Join(steamPath, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	vdf := `"users"
{
	"76561198000000001"
	{
		"PersonaName"		"Alpha"
	}
}`
	if err := os.WriteFile(filepath.Join(steamPath, "config", "loginusers.vdf"), []byte(vdf), 0o644); err != nil {
		t.Fatal(err)
	}
	m.steamPathOverride = steamPath

	profiles, err := m.ListSteamProfiles()
	if err != nil {
		t.Fatalf("list steam profiles: %v", err)
	}
	want := []SteamProfile{{SteamID: "76561198000000001", DisplayName: "Alpha"}, {SteamID: "76561198000000002", DisplayName: "76561198000000002"}}
	if !reflect.DeepEqual(profiles, want) {
		t.Fatalf("expected %v, got %v", want, profiles)
	}
}

func TestListSteamProfilesFallsBackWhenLoginUsersMissing(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	m, err := New(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	m.SaveRoot = filepath.Join(baseDir, "saves")
	if err := os.MkdirAll(filepath.Join(m.SaveRoot, "76561198000000003"), 0o755); err != nil {
		t.Fatal(err)
	}
	m.steamPathOverride = filepath.Join(baseDir, "Steam")

	profiles, err := m.ListSteamProfiles()
	if err != nil {
		t.Fatalf("list steam profiles: %v", err)
	}
	want := []SteamProfile{{SteamID: "76561198000000003", DisplayName: "76561198000000003"}}
	if !reflect.DeepEqual(profiles, want) {
		t.Fatalf("expected %v, got %v", want, profiles)
	}
}
