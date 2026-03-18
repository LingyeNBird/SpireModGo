//go:build windows

package manager

import (
	"strings"

	"golang.org/x/sys/windows/registry"
)

func readSteamPathFromRegistry() string {
	keys := []string{
		`SOFTWARE\Valve\Steam`,
		`SOFTWARE\WOW6432Node\Valve\Steam`,
	}
	valueNames := []string{"SteamPath", "InstallPath"}
	for _, keyPath := range keys {
		for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
			key, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			for _, valueName := range valueNames {
				value, _, err := key.GetStringValue(valueName)
				if err == nil && strings.TrimSpace(value) != "" {
					_ = key.Close()
					return strings.ReplaceAll(value, "/", `\`)
				}
			}
			_ = key.Close()
		}
	}
	return ""
}
