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
	installedState := t("Game directory is not configured yet. Open Settings before install or uninstall actions.")
	if installedMods, err := app.state.ListInstalledMods(); err == nil {
		installedCount = len(installedMods)
		installedState = fmt.Sprintf("%d installed mod folder(s).", installedCount)
	} else {
		installedState = app.localizeError(err)
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
		summaryText.WriteString(t("Game directory: (not configured)\n"))
	} else {
		summaryText.WriteString(t("Game directory: %s\n", app.state.GameDir()))
	}
	summaryText.WriteString(t("Mod source: %s\n", app.manager.ModsSource))
	summaryText.WriteString(t("Save root: %s\n\n", app.manager.SaveRoot))

	if availableErr != nil {
		summaryText.WriteString(t("Available packages: %s\n", app.localizeError(availableErr)))
	} else {
		summaryText.WriteString(t("Available packages: %d\n", len(availableMods)))
	}
	summaryText.WriteString(t("Installed mods: %s\n", installedState))

	if steamErr != nil {
		summaryText.WriteString(t("Steam profiles: %s\n", app.localizeError(steamErr)))
	} else {
		summaryText.WriteString(t("Steam profiles: %d\n", len(steamIDs)))
		if app.state.SelectedSteamID() != "" {
			summaryText.WriteString(t("Active Steam profile: %s\n", app.state.SelectedSteamID()))
			summaryText.WriteString(t("Available backups: %d\n", backupCount))
		}
	}
	s.summary = summaryText.String()
	s.notes = strings.Join([]string{
		t("What This Build Includes"),
		"",
		t("- Install mods from the bundled Mods packages."),
		t("- Uninstall selected mod folders or wipe the whole mods directory."),
		t("- Inspect installed mod manifests and copied files."),
		t("- Copy saves between vanilla and modded slots, plus backup and restore."),
		t("- Manage local settings, game-path detection, and .bak cleanup."),
		"",
		t("Notes"),
		"",
		t("- Destructive actions always ask for confirmation first."),
		t("- Operation results stay in the bottom status/log pane."),
		t("- Online update and self-update features are intentionally omitted."),
	}, "\n")
}

func (s *homeScreen) view(app *appModel, width, height int) string {
	leftWidth, _ := splitContentWidths(width, 28, 24)
	return renderSplitBody(t("Overview"), s.summary, t("Capabilities"), s.notes, width, height, leftWidth)
}

func (s *homeScreen) help() string {
	return t("Home: use the sidebar to switch pages")
}
