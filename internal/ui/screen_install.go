package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"spiremodgo/internal/manager"
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

func (s *installScreen) handleMouse(app *appModel, msg tea.MouseMsg, x, y, width, height int) tea.Cmd {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil
	}
	leftWidth, _ := splitContentWidths(width, 28, 24)
	layout := newSplitBodyLayout(width, height, leftWidth)
	if layout.leftBody.contains(x, y) {
		localY := y - layout.leftBody.y
		switch {
		case localY == 0:
			chosen := make([]manager.ModPackage, 0)
			for _, mod := range s.mods {
				if s.selected[mod.InstallName] {
					chosen = append(chosen, mod)
				}
			}
			s.install(app, chosen)
		case localY == 1:
			s.install(app, s.mods)
		case localY >= 3:
			idx := localY - 3
			if idx >= 0 && idx < len(s.mods) {
				s.cursor = idx
				key := s.mods[idx].InstallName
				s.selected[key] = !s.selected[key]
			}
		}
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
		return t("Failed to load available packages:\n\n%s", s.err)
	}
	if len(s.mods) == 0 {
		return t("No packages were found in the bundled Mods directory.")
	}
	items := make([]string, 0, len(s.mods))
	for _, mod := range s.mods {
		items = append(items, renderModListLabel(mod, s.selected[mod.InstallName]))
	}
	leftWidth, _ := splitContentWidths(width, 28, 24)
	leftBody := strings.Join([]string{
		renderActionLine(t("Install Selected"), false),
		renderActionLine(t("Install All"), false),
		"",
		renderList(items, s.cursor, app.focus == focusContent),
	}, "\n")
	return renderSplitBody(t("Packages"), leftBody, t("Package Details"), renderAvailableModDetail(s.mods[s.cursor]), width, height, leftWidth)
}

func (s *installScreen) help() helpSection {
	return helpSection{
		Title: t("Install:"),
		Items: []helpItem{
			{Action: t("Click"), Description: t("package rows or action buttons")},
			{Action: t("up/down"), Description: t("move")},
			{Action: t("space"), Description: t("toggle")},
			{Action: t("i"), Description: t("install selected")},
			{Action: t("a"), Description: t("install all")},
		},
	}
}
