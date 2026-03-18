//go:build !windows

package manager

func readSteamPathFromRegistry() string {
	return ""
}
