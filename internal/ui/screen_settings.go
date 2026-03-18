package ui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type settingsScreen struct {
	input        textinput.Model
	actionCursor int
	editing      bool
	summary      string
}

func (s *settingsScreen) init() {
	input := textinput.New()
	input.Placeholder = t("Enter SlayTheSpire2 directory")
	input.Width = 48
	s.input = input
}

func (s *settingsScreen) refresh(app *appModel) {
	if !s.editing {
		s.input.SetValue(app.state.GameDir())
	}
	s.input.Placeholder = t("Enter SlayTheSpire2 directory")
	steamIDs, steamErr := app.state.ListSteamIDs()
	backupCount := 0
	if steamErr == nil && app.state.SelectedSteamID() != "" {
		if backups, err := app.state.ListBackups(); err == nil {
			backupCount = len(backups)
		}
	}
	bakCount := 0
	if dir := app.state.GameDir(); dir != "" {
		bakCount = countBakFiles(filepath.Join(dir, "mods"))
	}
	lines := []string{
		t("Current Paths"),
		"",
		t("Config file: %s", app.manager.ConfigPath),
		t("Bundled Mods: %s", app.manager.ModsSource),
		t("Save root: %s", app.manager.SaveRoot),
	}
	if app.state.GameDir() != "" {
		lines = append(lines, t("Game dir: %s", app.state.GameDir()))
	} else {
		lines = append(lines, t("Game dir: (not configured)"))
	}
	lines = append(lines, "")
	if steamErr != nil {
		lines = append(lines, t("Steam profiles: %s", app.localizeError(steamErr)))
	} else {
		lines = append(lines, t("Steam profiles: %d", len(steamIDs)))
		if app.state.SelectedSteamID() != "" {
			lines = append(lines, t("Active profile: %s", app.state.SelectedSteamID()))
			lines = append(lines, t("Backups for active profile: %d", backupCount))
		}
	}
	lines = append(lines, t("Detected .bak files in game mods: %d", bakCount))
	s.summary = strings.Join(lines, "\n")
}

func (s *settingsScreen) handleKey(app *appModel, msg tea.KeyMsg) tea.Cmd {
	if s.editing {
		switch msg.String() {
		case "enter":
			s.editing = false
			return nil
		case "esc":
			s.editing = false
			return nil
		}
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		return cmd
	}
	switch msg.String() {
	case "up", "k":
		if s.actionCursor > 0 {
			s.actionCursor--
		}
	case "down", "j":
		if s.actionCursor < 4 {
			s.actionCursor++
		}
	case "e":
		s.editing = true
		s.input.Focus()
	case "enter":
		s.runAction(app)
	}
	return nil
}

func (s *settingsScreen) runAction(app *appModel) {
	switch s.actionCursor {
	case 0:
		dir, err := app.state.AutoDetectGameDir()
		if err != nil {
			app.showError("Auto detect failed", err)
			return
		}
		s.input.SetValue(dir)
		s.refresh(app)
		app.logSuccess("Detected and saved game directory: %s", dir)
	case 1:
		if err := app.state.SetGameDir(s.input.Value()); err != nil {
			app.showError("Save path failed", err)
			return
		}
		s.refresh(app)
		app.logSuccess("Saved game directory: %s", s.input.Value())
	case 2:
		app.showConfirm("Clear Config", "Delete the saved game-directory config file and reset the current path?", func(model *appModel) {
			if err := model.state.ClearConfig(); err != nil {
				model.showError("Clear config failed", err)
				return
			}
			s.input.SetValue("")
			s.refresh(model)
			model.logWarn("Cleared the saved config file")
		})
	case 3:
		app.showConfirm("Cleanup .bak Files", "Delete all .bak files under the game's mods directory?", func(model *appModel) {
			removed, err := model.state.CleanupBakFiles()
			if err != nil {
				model.showError("Cleanup failed", err)
				return
			}
			if len(removed) == 0 {
				model.logInfo("No .bak files were found to clean up")
			} else {
				model.logSuccess("Removed %d .bak file(s)", len(removed))
			}
			s.refresh(model)
		})
	case 4:
		s.refresh(app)
		app.logInfo("Refreshed settings summary")
	}
}

func (s *settingsScreen) view(app *appModel, width, height int) string {
	actions := []string{t("Auto Detect"), t("Save Path"), t("Clear Config"), t("Cleanup .bak"), t("Refresh")}
	if s.editing {
		s.input.Prompt = "> "
	} else {
		s.input.Blur()
		s.input.Prompt = "  "
	}
	actionText := renderList(actions, s.actionCursor, app.focus == focusContent && !s.editing)
	leftWidth, rightWidth := splitContentWidths(width, 28, 24)
	leftBody := strings.Join([]string{
		t("Game Dir"),
		s.input.View(),
		"",
		t("Actions"),
		actionText,
		"",
		mutedStyle.Render(t("Press e to edit the path input.")),
	}, "\n")
	left := renderFlatColumn(t("Actions"), leftBody, leftWidth, height)
	right := renderFlatColumn(t("Current Paths and State"), s.summary, rightWidth, height)
	return joinFlatColumns(left, right, leftWidth, rightWidth)
}

func (s *settingsScreen) help() string {
	if s.editing {
		return t("Settings: type path | enter finish edit | esc cancel edit")
	}
	return t("Settings: up/down action | e edit path | enter run action")
}
