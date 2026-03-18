package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"slaymodgo/internal/manager"
)

type installedScreen struct {
	mods   []manager.InstalledMod
	cursor int
	err    string
}

func (s *installedScreen) refresh(app *appModel) {
	mods, err := app.state.ListInstalledMods()
	s.mods = mods
	s.err = ""
	if err != nil {
		s.err = app.localizeError(err)
	}
	if s.cursor >= len(s.mods) {
		s.cursor = maxInt(0, len(s.mods)-1)
	}
}

func (s *installedScreen) handleKey(app *appModel, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if s.cursor > 0 {
			s.cursor--
		}
	case "down", "j":
		if s.cursor < len(s.mods)-1 {
			s.cursor++
		}
	}
	return nil
}

func (s *installedScreen) view(app *appModel, width, height int) string {
	if s.err != "" {
		return renderFlatColumn(t("Status"), s.err+"\n\n"+t("Open Settings to configure the game path before inspecting installed mods."), width, height)
	}
	if len(s.mods) == 0 {
		return renderFlatColumn(t("Status"), t("No installed mods were found in the game's mods directory."), width, height)
	}
	items := make([]string, 0, len(s.mods))
	for _, mod := range s.mods {
		items = append(items, mod.Label)
	}
	leftWidth, rightWidth := splitContentWidths(width, 28, 24)
	left := renderFlatColumn(t("Installed Mods"), renderList(items, s.cursor, app.focus == focusContent), leftWidth, height)
	right := renderFlatColumn(t("Mod Details"), renderInstalledModDetail(s.mods[s.cursor]), rightWidth, height)
	return joinFlatColumns(left, right, leftWidth, rightWidth)
}

func (s *installedScreen) help() string {
	return t("Installed: up/down move")
}
