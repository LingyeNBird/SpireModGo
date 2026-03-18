package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"slaymodgo/internal/manager"
)

type installScreen struct {
	mods     []manager.ModPackage
	selected map[string]bool
	cursor   int
	err      string
}

func (s *installScreen) refresh(app *appModel) {
	mods, err := app.state.ListAvailableMods()
	s.mods = mods
	s.err = ""
	if err != nil {
		s.err = app.localizeError(err)
	}
	if s.selected == nil {
		s.selected = map[string]bool{}
	}
	if s.cursor >= len(s.mods) {
		s.cursor = maxInt(0, len(s.mods)-1)
	}
}

func (s *installScreen) handleKey(app *appModel, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if s.cursor > 0 {
			s.cursor--
		}
	case "down", "j":
		if s.cursor < len(s.mods)-1 {
			s.cursor++
		}
	case " ", "enter":
		if len(s.mods) > 0 {
			key := s.mods[s.cursor].InstallName
			s.selected[key] = !s.selected[key]
		}
	case "a":
		s.install(app, s.mods)
	case "i":
		chosen := make([]manager.ModPackage, 0)
		for _, mod := range s.mods {
			if s.selected[mod.InstallName] {
				chosen = append(chosen, mod)
			}
		}
		s.install(app, chosen)
	}
	return nil
}

func (s *installScreen) install(app *appModel, items []manager.ModPackage) {
	if len(items) == 0 {
		app.showInfo("Nothing Selected", "Select one or more packages before installing.")
		return
	}
	results, err := app.state.InstallMods(items)
	if err != nil {
		app.showError("Install failed", err)
		return
	}
	for _, result := range results {
		app.logSuccess("Installed %s (%d file(s) copied)", result.Mod.InstallName, result.FilesCopied)
		if result.EnableChanged {
			app.logInfo("Enabled the in-game mod toggle in settings.save where needed")
		}
		for _, file := range result.Files {
			if file.Err != nil {
				app.logError("%s: %v", file.Name, file.Err)
				continue
			}
			if file.Replaced {
				app.logWarn("Replaced locked target for %s and kept backup %s", file.Name, file.BackupName)
			}
		}
	}
	s.selected = map[string]bool{}
	s.refresh(app)
	app.offerPostInstallSaveCopy()
}

func (s *installScreen) view(app *appModel, width, height int) string {
	if s.err != "" {
		return renderFlatColumn(t("Status"), t("Failed to load available packages:\n\n%s", s.err), width, height)
	}
	if len(s.mods) == 0 {
		return renderFlatColumn(t("Status"), t("No packages were found in the bundled Mods directory."), width, height)
	}
	items := make([]string, 0, len(s.mods))
	for _, mod := range s.mods {
		items = append(items, renderModListLabel(mod, s.selected[mod.InstallName]))
	}
	leftWidth, rightWidth := splitContentWidths(width, 28, 24)
	left := renderFlatColumn(t("Available Packages"), renderList(items, s.cursor, app.focus == focusContent), leftWidth, height)
	right := renderFlatColumn(t("Package Details"), renderAvailableModDetail(s.mods[s.cursor]), rightWidth, height)
	return joinFlatColumns(left, right, leftWidth, rightWidth)
}

func (s *installScreen) help() string {
	return t("Install: up/down move | space toggle | i install selected | a install all")
}
