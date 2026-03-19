package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"spiremodgo/internal/manager"
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
		s.err = app.localizeError(err)
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

func (s *uninstallScreen) handleMouse(app *appModel, msg tea.MouseMsg, x, y, width, height int) tea.Cmd {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil
	}
	leftWidth, _ := splitContentWidths(width, 28, 24)
	layout := newSplitBodyLayout(width, height, leftWidth)
	if layout.leftBody.contains(x, y) {
		localY := y - layout.leftBody.y
		switch {
		case localY == 0:
			s.uninstallSelected(app)
		case localY == 1:
			s.uninstallAll(app)
		case localY >= 3:
			idx := localY - 3
			if idx >= 0 && idx < len(s.mods) {
				s.cursor = idx
				key := s.mods[idx].DirName
				s.selected[key] = !s.selected[key]
			}
		}
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
	app.showConfirm("Confirm Uninstall", t("Remove the selected installed mod folder(s)?\n\n%s", label), func(model *appModel) {
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
		return s.err + "\n\n" + t("Open Settings to configure the game path before uninstalling.")
	}
	if len(s.mods) == 0 {
		return t("No installed mods were found in the game's mods directory.")
	}
	items := make([]string, 0, len(s.mods))
	for _, mod := range s.mods {
		items = append(items, renderInstalledModListLabel(mod, s.selected[mod.DirName]))
	}
	leftWidth, _ := splitContentWidths(width, 28, 24)
	leftBody := strings.Join([]string{
		renderActionLine(t("Remove Selected"), false),
		renderActionLine(t("Remove All"), false),
		"",
		renderList(items, s.cursor, app.focus == focusContent),
	}, "\n")
	return renderSplitBody(t("Installed Mods"), leftBody, t("Removal Details"), renderInstalledModDetail(s.mods[s.cursor]), width, height, leftWidth)
}

func (s *uninstallScreen) help() helpSection {
	return helpSection{
		Title: t("Uninstall:"),
		Items: []helpItem{
			{Action: t("Click"), Description: t("rows or action buttons")},
			{Action: t("up/down"), Description: t("move")},
			{Action: t("space"), Description: t("toggle")},
			{Action: t("d"), Description: t("remove selected")},
			{Action: t("x"), Description: t("remove all")},
		},
	}
}
