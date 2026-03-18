package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"slaymodgo/internal/manager"
)

type uninstallScreen struct {
	mods     []manager.InstalledMod
	selected map[string]bool
	cursor   int
	err      string
}

func (s *uninstallScreen) refresh(app *appModel) {
	mods, err := app.state.ListInstalledMods()
	s.mods = mods
	s.err = ""
	if err != nil {
		s.err = err.Error()
	}
	if s.selected == nil {
		s.selected = map[string]bool{}
	}
	if s.cursor >= len(s.mods) {
		s.cursor = maxInt(0, len(s.mods)-1)
	}
}

func (s *uninstallScreen) handleKey(app *appModel, msg tea.KeyMsg) tea.Cmd {
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
			key := s.mods[s.cursor].DirName
			s.selected[key] = !s.selected[key]
		}
	case "d":
		s.uninstallSelected(app)
	case "x":
		s.uninstallAll(app)
	}
	return nil
}

func (s *uninstallScreen) uninstallSelected(app *appModel) {
	names := make([]string, 0)
	for _, mod := range s.mods {
		if s.selected[mod.DirName] {
			names = append(names, mod.DirName)
		}
	}
	if len(names) == 0 {
		app.showInfo("Nothing Selected", "Select one or more installed mods before uninstalling.")
		return
	}
	label := strings.Join(names, ", ")
	app.showConfirm("Confirm Uninstall", "Remove the selected installed mod folder(s)?\n\n"+label, func(model *appModel) {
		results, warn, err := model.state.UninstallMods(names)
		if err != nil {
			model.showError("Uninstall failed", err)
			return
		}
		for _, result := range results {
			if result.Err != nil {
				model.logError("Failed to remove %s: %v", result.Name, result.Err)
				continue
			}
			model.logSuccess("Removed installed mod folder %s", result.Name)
		}
		if warn {
			model.logWarn("No installed mods remain. Disable mods manually in the game settings if you want to return to vanilla saves.")
		}
		s.selected = map[string]bool{}
		s.refresh(model)
	})
}

func (s *uninstallScreen) uninstallAll(app *appModel) {
	if len(s.mods) == 0 {
		app.showInfo("Nothing Installed", "There are no installed mods to remove.")
		return
	}
	app.showConfirm("Confirm Full Uninstall", "This will remove every folder under the game's mods directory. Continue?", func(model *appModel) {
		warn, err := model.state.UninstallAllMods()
		if err != nil {
			model.showError("Uninstall all failed", err)
			return
		}
		model.logSuccess("Cleared the game's mods directory")
		if warn {
			model.logWarn("All mods were removed. Disable mods manually in the game settings if you want to switch back to vanilla saves.")
		}
		s.selected = map[string]bool{}
		s.refresh(model)
	})
}

func (s *uninstallScreen) view(app *appModel, width, height int) string {
	if s.err != "" {
		return renderFlatColumn("Status", s.err+"\n\nOpen Settings to configure the game path before uninstalling.", width, height)
	}
	if len(s.mods) == 0 {
		return renderFlatColumn("Status", "No installed mods were found in the game's mods directory.", width, height)
	}
	items := make([]string, 0, len(s.mods))
	for _, mod := range s.mods {
		items = append(items, renderInstalledModListLabel(mod, s.selected[mod.DirName]))
	}
	leftWidth, rightWidth := splitContentWidths(width, 28, 24)
	left := renderFlatColumn("Installed Mods", renderList(items, s.cursor, app.focus == focusContent), leftWidth, height)
	right := renderFlatColumn("Removal Details", renderInstalledModDetail(s.mods[s.cursor]), rightWidth, height)
	return joinFlatColumns(left, right, leftWidth, rightWidth)
}

func (s *uninstallScreen) help() string {
	return "Uninstall: up/down move | space toggle | d remove selected | x remove all"
}
