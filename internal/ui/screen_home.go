package ui

import (
	"fmt"
	"strings"
)

type homeScreen struct {
	summary string
	notes   string
}

func (s *homeScreen) refresh(app *appModel) {
	availableMods, availableErr := app.state.ListAvailableMods()
	installedCount := 0
	installedState := "Unavailable until the game directory is configured."
	if installedMods, err := app.state.ListInstalledMods(); err == nil {
		installedCount = len(installedMods)
		installedState = fmt.Sprintf("%d installed mod folder(s).", installedCount)
	} else {
		installedState = err.Error()
	}

	steamIDs, steamErr := app.state.ListSteamIDs()
	backupCount := 0
	if steamErr == nil && app.state.SelectedSteamID() != "" {
		if backups, err := app.state.ListBackups(); err == nil {
			backupCount = len(backups)
		}
	}

	var summaryText strings.Builder
	if app.state.GameDir() == "" {
		summaryText.WriteString("Game directory: (not configured)\n")
	} else {
		summaryText.WriteString(fmt.Sprintf("Game directory: %s\n", app.state.GameDir()))
	}
	summaryText.WriteString(fmt.Sprintf("Mod source: %s\n", app.manager.ModsSource))
	summaryText.WriteString(fmt.Sprintf("Save root: %s\n\n", app.manager.SaveRoot))

	if availableErr != nil {
		summaryText.WriteString(fmt.Sprintf("Available packages: %s\n", availableErr.Error()))
	} else {
		summaryText.WriteString(fmt.Sprintf("Available packages: %d\n", len(availableMods)))
	}
	summaryText.WriteString(fmt.Sprintf("Installed mods: %s\n", installedState))

	if steamErr != nil {
		summaryText.WriteString(fmt.Sprintf("Steam profiles: %s\n", steamErr.Error()))
	} else {
		summaryText.WriteString(fmt.Sprintf("Steam profiles: %d\n", len(steamIDs)))
		if app.state.SelectedSteamID() != "" {
			summaryText.WriteString(fmt.Sprintf("Active Steam profile: %s\n", app.state.SelectedSteamID()))
			summaryText.WriteString(fmt.Sprintf("Available backups: %d\n", backupCount))
		}
	}
	s.summary = summaryText.String()
	s.notes = strings.Join([]string{
		"What This Build Includes",
		"",
		"- Install mods from the bundled Mods packages.",
		"- Uninstall selected mod folders or wipe the whole mods directory.",
		"- Inspect installed mod manifests and copied files.",
		"- Copy saves between vanilla and modded slots, plus backup and restore.",
		"- Manage local settings, game-path detection, and .bak cleanup.",
		"",
		"Notes",
		"",
		"- Destructive actions always ask for confirmation first.",
		"- Operation results stay in the bottom status/log pane.",
		"- Online update and self-update features are intentionally omitted.",
	}, "\n")
}

func (s *homeScreen) view(app *appModel, width, height int) string {
	leftWidth, rightWidth := splitContentWidths(width, 28, 24)
	left := renderFlatColumn("Overview", s.summary, leftWidth, height)
	right := renderFlatColumn("Capabilities", s.notes, rightWidth, height)
	return joinFlatColumns(left, right, leftWidth, rightWidth)
}

func (s *homeScreen) help() string {
	return "Home: use the sidebar to switch pages"
}
