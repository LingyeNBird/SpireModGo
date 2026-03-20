package manager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	githubRepoOwner = "LingyeNBird"
	githubRepoName  = "SpireModGo"
)

var latestReleaseAPIURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubRepoOwner, githubRepoName)

func releaseAssetName(goos, goarch string) string {
	return fmt.Sprintf("SpireModGo_%s_%s.exe", goos, goarch)
}

func selectReleaseAsset(assets []ReleaseAsset, goos, goarch string) (ReleaseAsset, bool) {
	expected := strings.ToLower(releaseAssetName(goos, goarch))
	for _, asset := range assets {
		if strings.ToLower(strings.TrimSpace(asset.Name)) == expected {
			return asset, true
		}
	}
	return ReleaseAsset{}, false
}

func (m *Manager) CheckForUpdates() (UpdateCheckResult, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(latestReleaseAPIURL)
	if err != nil {
		return UpdateCheckResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return UpdateCheckResult{}, fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return UpdateCheckResult{}, err
	}

	result := UpdateCheckResult{
		CurrentVersion: AppVersion,
		LatestVersion:  strings.TrimSpace(release.TagName),
		ReleaseURL:     strings.TrimSpace(release.HTMLURL),
	}

	if asset, ok := selectReleaseAsset(release.Assets, runtime.GOOS, runtime.GOARCH); ok {
		result.AssetName = asset.Name
		result.AssetURL = strings.TrimSpace(asset.BrowserDownloadURL)
	}

	if result.LatestVersion == "" {
		return result, fmt.Errorf("latest release has empty tag_name")
	}
	result.HasUpdate = compareVersionStrings(result.CurrentVersion, result.LatestVersion) < 0
	return result, nil
}

func (m *Manager) OpenURL(url string) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("empty url")
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
