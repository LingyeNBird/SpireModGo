package ui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

const settingsInputHeight = 3

type settingsScreen struct {
	input        textarea.Model
	actionCursor int
	editing      bool
	summary      string
}

func (s *settingsScreen) init() {
	input := textarea.New()
	input.Placeholder = t("Enter SlayTheSpire2 directory")
	input.ShowLineNumbers = false
	input.Prompt = ""
	input.SetHeight(settingsInputHeight)
	input.SetWidth(48)
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
		t("Bundled Mods: %s", app.manager.DisplayAvailableModsRoot()),
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
			s.input.Blur()
			return nil
		case "esc":
			s.input.SetValue(app.state.GameDir())
			s.editing = false
			s.input.Blur()
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

func (s *settingsScreen) handleMouse(app *appModel, msg tea.MouseMsg, x, y, width, height int) tea.Cmd {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil
	}
	leftWidth := maxInt(1, (width-3)/2)
	layout := newSplitBodyLayout(width, height, leftWidth)
	inputStart := 1
	inputEnd := inputStart + settingsInputHeight - 1
	editActionY := inputEnd + 2
	actionStartY := editActionY + 1
	if !layout.leftBody.contains(x, y) {
		return nil
	}
	localY := y - layout.leftBody.y
	switch {
	case localY >= inputStart && localY <= inputEnd:
		s.editing = true
		s.input.Focus()
	case localY == editActionY:
		s.editing = true
		s.input.Focus()
	case localY >= actionStartY && localY < actionStartY+5:
		s.actionCursor = localY - actionStartY
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
	leftWidth := maxInt(1, (width-3)/2)
	layout := newSplitBodyLayout(width, height, leftWidth)
	if s.editing {
		s.input.Prompt = "> "
		_ = s.input.Focus()
	} else {
		s.input.Blur()
		s.input.Prompt = "  "
	}
	s.input.Placeholder = t("Enter SlayTheSpire2 directory")
	s.input.SetHeight(settingsInputHeight)
	s.input.SetWidth(layout.leftBody.width)
	actionText := renderActionButtonList(actions, s.actionCursor, app.focus == focusContent && !s.editing)
	leftBody := strings.Join([]string{
		t("Game Dir"),
		s.input.View(),
		"",
		renderInlineButton(t("Edit Path"), s.editing, app.focus == focusContent && s.editing),
		actionText,
		"",
		mutedStyle.Render(t("Press e to edit the path input.")),
	}, "\n")
	summary := wrapBodyText(s.summary, layout.rightBody.width)
	return renderSplitBody(t("Actions"), leftBody, t("Current Paths and State"), summary, width, height, leftWidth)
}

func (s *settingsScreen) help() helpSection {
	if s.editing {
		return helpSection{
			Title: t("Settings:"),
			Items: []helpItem{
				{Action: t("Click"), Description: t("or type path")},
				{Action: t("Enter"), Description: t("finish edit")},
				{Action: t("Esc"), Description: t("cancel edit")},
			},
		}
	}
	return helpSection{
		Title: t("Settings:"),
		Items: []helpItem{
			{Action: t("Click"), Description: t("path or actions")},
			{Action: t("up/down"), Description: t("action")},
			{Action: t("e"), Description: t("edit path")},
			{Action: t("Enter"), Description: t("run action")},
		},
	}
}
