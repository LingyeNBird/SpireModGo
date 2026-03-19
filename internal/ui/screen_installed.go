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

func (s *installedScreen) handleMouse(app *appModel, msg tea.MouseMsg, x, y, width, height int) tea.Cmd {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil
	}
	leftWidth, _ := splitContentWidths(width, 28, 24)
	layout := newSplitBodyLayout(width, height, leftWidth)
	if layout.leftBody.contains(x, y) {
		idx := y - layout.leftBody.y
		if idx >= 0 && idx < len(s.mods) {
			s.cursor = idx
		}
	}
	return nil
}

func (s *installedScreen) view(app *appModel, width, height int) string {
	if s.err != "" {
		return s.err + "\n\n" + t("Open Settings to configure the game path before inspecting installed mods.")
	}
	if len(s.mods) == 0 {
		return t("No installed mods were found in the game's mods directory.")
	}
	items := make([]string, 0, len(s.mods))
	for _, mod := range s.mods {
		items = append(items, mod.Label)
	}
	leftWidth, _ := splitContentWidths(width, 28, 24)
	return renderSplitBody(t("Installed Mods"), renderList(items, s.cursor, app.focus == focusContent), t("Mod Details"), renderInstalledModDetail(s.mods[s.cursor]), width, height, leftWidth)
}

func (s *installedScreen) help() string {
	return t("Installed: click rows or use up/down move")
}
