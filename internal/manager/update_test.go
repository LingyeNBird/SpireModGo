package manager

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func TestReleaseAssetName(t *testing.T) {
	if got := releaseAssetName("windows", "arm64"); got != "SpireModGo_windows_arm64.exe" {
		t.Fatalf("unexpected release asset name: %s", got)
	}
}

func TestSelectReleaseAssetMatchesCurrentArch(t *testing.T) {
	asset, ok := selectReleaseAsset([]ReleaseAsset{{Name: "SpireModGo_windows_386.exe"}, {Name: "SpireModGo_windows_amd64.exe"}}, "windows", "amd64")
	if !ok {
		t.Fatal("expected asset match for amd64")
	}
	if asset.Name != "SpireModGo_windows_amd64.exe" {
		t.Fatalf("unexpected asset selected: %s", asset.Name)
	}
}

func TestCheckForUpdatesDetectsNewerRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assetName := releaseAssetName(runtime.GOOS, runtime.GOARCH)
		_, _ = fmt.Fprintf(w, `{"tag_name":"v1.0.1","html_url":"https://github.com/LingyeNBird/SpireModGo/releases/tag/v1.0.1","assets":[{"name":%q,"browser_download_url":"https://example.com/%s"}]}`, assetName, assetName)
	}))
	defer server.Close()

	previous := latestReleaseAPIURL
	latestReleaseAPIURL = server.URL
	defer func() { latestReleaseAPIURL = previous }()

	m, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	result, err := m.CheckForUpdates()
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasUpdate {
		t.Fatalf("expected update to be available, got %+v", result)
	}
	if result.CurrentVersion != AppVersion {
		t.Fatalf("expected current version %s, got %s", AppVersion, result.CurrentVersion)
	}
	if result.LatestVersion != "v1.0.1" {
		t.Fatalf("unexpected latest version: %s", result.LatestVersion)
	}
	if result.ReleaseURL == "" {
		t.Fatal("expected release page url")
	}
	if result.AssetName == "" || result.AssetURL == "" {
		t.Fatalf("expected matching asset, got %+v", result)
	}
}

func TestCheckForUpdatesReturnsNoUpdateForSameVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"tag_name":%q,"html_url":"https://github.com/LingyeNBird/SpireModGo/releases/tag/%s","assets":[]}`, AppVersion, AppVersion)
	}))
	defer server.Close()

	previous := latestReleaseAPIURL
	latestReleaseAPIURL = server.URL
	defer func() { latestReleaseAPIURL = previous }()

	m, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	result, err := m.CheckForUpdates()
	if err != nil {
		t.Fatal(err)
	}
	if result.HasUpdate {
		t.Fatalf("expected no update, got %+v", result)
	}
}

func TestCheckForUpdatesFallsBackToReleasePageWhenAssetMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"tag_name":"v1.2.0","html_url":"https://github.com/LingyeNBird/SpireModGo/releases/tag/v1.2.0","assets":[{"name":"SpireModGo_notmatching_asset.exe","browser_download_url":"https://example.com/SpireModGo_notmatching_asset.exe"}]}`)
	}))
	defer server.Close()

	previous := latestReleaseAPIURL
	latestReleaseAPIURL = server.URL
	defer func() { latestReleaseAPIURL = previous }()

	m, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	result, err := m.CheckForUpdates()
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasUpdate {
		t.Fatalf("expected update to still be available, got %+v", result)
	}
	if result.ReleaseURL == "" {
		t.Fatal("expected release page fallback url")
	}
	if result.AssetURL != "" {
		t.Fatalf("expected no asset url for fallback case, got %+v", result)
	}
}
