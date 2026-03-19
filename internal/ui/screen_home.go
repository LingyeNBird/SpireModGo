package ui

import (
	"strings"
)

type homeScreen struct {
	overview string
	guide    string
}

func (s *homeScreen) refresh(app *appModel) {
	availableMods, availableErr := app.state.ListAvailableMods()
	installedCount := 0
	installedState := t("Game directory is not configured yet. Open Settings before install or uninstall actions.")
	if installedMods, err := app.state.ListInstalledMods(); err == nil {
		installedCount = len(installedMods)
		installedState = t("%d installed mod folder(s).", installedCount)
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
	summaryText.WriteString(t("Mod directory: %s\n", app.manager.ModsSource))
	summaryText.WriteString(t("Save root: %s\n", app.manager.SaveRoot))
	summaryText.WriteString(t("Installed mods: %s\n", installedState))

	if availableErr != nil {
		summaryText.WriteString(t("Available packages: %s\n", app.localizeError(availableErr)))
	} else {
		summaryText.WriteString(t("Available packages: %d\n", len(availableMods)))
	}

	if steamErr != nil {
		summaryText.WriteString(t("Steam profiles: %s\n", app.localizeError(steamErr)))
	} else {
		summaryText.WriteString(t("Steam profiles: %d\n", len(steamIDs)))
		if app.state.SelectedSteamID() != "" {
			summaryText.WriteString(t("Active Steam profile: %s\n", app.state.SelectedSteamID()))
			summaryText.WriteString(t("Available backups: %d\n", backupCount))
		}
	}
	s.overview = summaryText.String()
	s.guide = strings.Join([]string{
		t("This build focuses on local mod, save, and settings workflows."),
		"",
		t("- Mod Management combines install, uninstall, and detail inspection."),
		t("- Save Management shows vanilla and modded slots together."),
		t("- Copy saves now picks the destination slot in a dedicated popup."),
		t("- Settings still manages game-path detection and .bak cleanup."),
		"",
		t("- Destructive actions always ask for confirmation first."),
		t("- Operation results stay in the bottom status/log pane."),
		t("- Online update and self-update features are intentionally omitted."),
	}, "\n")
}

func (s *homeScreen) view(app *appModel, width, height int) string {
	leftWidth := maxInt(1, (width-3)/2)
	layout := newSplitBodyLayout(width, height, leftWidth)
	overview := wrapBodyText(s.overview, layout.leftBody.width)
	guide := wrapBodyText(s.guide, layout.rightBody.width)
	return renderSplitBody(t("Information Overview"), overview, t("Feature Guide"), guide, width, height, leftWidth)
}

func (s *homeScreen) help() string {
	return t("Home: use the sidebar to switch pages")
}
