package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"slaymodgo/internal/manager"
)

type savesScreen struct {
	listCursor     int
	lastSlotCursor int
	backupCursor   int
	section        int
	steamIDs       []string
	normalSlots    []manager.SaveSlotInfo
	moddedSlots    []manager.SaveSlotInfo
	backups        []manager.BackupInfo
	err            string
}

func (s *savesScreen) refresh(app *appModel) {
	if s.listCursor == 0 {
		s.listCursor = 1
	}
	if s.lastSlotCursor == 0 {
		s.lastSlotCursor = 1
	}
	s.err = ""
	ids, err := app.state.ListSteamIDs()
	s.steamIDs = ids
	if err != nil {
		s.err = app.localizeError(err)
		return
	}
	if len(ids) == 0 {
		return
	}
	normalSlots, err := app.state.ListSaveSlots(manager.SaveTypeNormal)
	if err != nil {
		s.err = app.localizeError(err)
		return
	}
	moddedSlots, err := app.state.ListSaveSlots(manager.SaveTypeModded)
	if err != nil {
		s.err = app.localizeError(err)
		return
	}
	backups, err := app.state.ListBackups()
	if err != nil {
		s.err = app.localizeError(err)
		return
	}
	s.normalSlots = normalSlots
	s.moddedSlots = moddedSlots
	s.backups = backups
	if s.listCursor > 6 {
		s.listCursor = 6
	}
	filtered := s.selectedBackups()
	if s.backupCursor >= len(filtered) {
		s.backupCursor = maxInt(0, len(filtered)-1)
	}
}

func (s *savesScreen) handleKey(app *appModel, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "s", "tab":
		s.section = 1 - s.section
	case "up", "k":
		if s.section == 0 {
			if s.listCursor > 0 {
				s.listCursor--
				if s.listCursor > 0 {
					s.lastSlotCursor = s.listCursor
				}
			}
		} else if s.backupCursor > 0 {
			s.backupCursor--
		}
	case "down", "j":
		if s.section == 0 {
			if s.listCursor < 6 {
				s.listCursor++
				s.lastSlotCursor = s.listCursor
			}
		} else if s.backupCursor < len(s.selectedBackups())-1 {
			s.backupCursor++
		}
	case "left", "h":
		if s.section == 0 && s.listCursor == 0 {
			s.switchSteamProfile(app, -1)
		}
	case "right", "l":
		if s.section == 0 && s.listCursor == 0 {
			s.switchSteamProfile(app, 1)
		}
	case "c":
		s.openCopyModal(app)
	case "b":
		s.backupSelected(app)
	case "r", "enter":
		if s.section == 1 {
			s.restoreSelected(app)
		}
	case "x":
		if s.section == 1 {
			s.deleteSelected(app)
		}
	}
	return nil
}

func (s *savesScreen) handleMouse(app *appModel, msg tea.MouseMsg, x, y, width, height int) tea.Cmd {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil
	}
	leftWidth := maxInt(1, (width-3)/2)
	layout := newSplitBodyLayout(width, height, leftWidth)
	if layout.leftBody.contains(x, y) {
		localX, localY := x-layout.leftBody.x, y-layout.leftBody.y
		s.section = 0
		switch localY {
		case 0:
			s.listCursor = 0
			if localX < layout.leftBody.width/2 {
				s.switchSteamProfile(app, -1)
			} else {
				s.switchSteamProfile(app, 1)
			}
		case 3, 4, 5:
			s.listCursor = localY - 2
			s.lastSlotCursor = s.listCursor
		case 8, 9, 10:
			s.listCursor = localY - 4
			s.lastSlotCursor = s.listCursor
		}
		return nil
	}
	if layout.rightBody.contains(x, y) {
		localX, localY := x-layout.rightBody.x, y-layout.rightBody.y
		filtered := s.selectedBackups()
		switch {
		case localY == 7:
			s.openCopyModal(app)
		case localY == 8:
			s.backupSelected(app)
		case localY == 10:
			if localX < layout.rightBody.width/2 {
				s.section = 1
				s.restoreSelected(app)
			} else {
				s.section = 1
				s.deleteSelected(app)
			}
		case localY >= 12:
			idx := localY - 12
			if idx >= 0 && idx < len(filtered) {
				s.section = 1
				s.backupCursor = idx
			}
		}
	}
	return nil
}

func (s *savesScreen) view(app *appModel, width, height int) string {
	if s.err != "" {
		return t("Save management error\n\n%s", s.err)
	}
	if len(s.steamIDs) == 0 {
		return t("No Steam save profiles were found under:\n\n%s\n\nLaunch the game once to create the save folders, then refresh this screen.", app.manager.SaveRoot)
	}
	leftWidth := maxInt(1, (width-3)/2)
	leftBody := s.renderLeftList(app, leftWidth)
	rightBody := s.renderRightPanel(app)
	return renderSplitBody(t("Save List"), leftBody, t("Save Management"), rightBody, width, height, leftWidth)
}

func (s *savesScreen) help() string {
	return t("Saves: left/right switch Steam profile | click slots and backups | c copy | b backup | r restore | x delete backup")
}

func (s *savesScreen) renderLeftList(app *appModel, width int) string {
	lines := []string{renderSelectableLine(renderValueControl(t("Steam ID"), app.state.SelectedSteamID()), s.listCursor == 0, app.focus == focusContent && s.section == 0), "", t("Vanilla Saves")}
	lines = append(lines, strings.Split(s.renderSaveSlotTable(manager.SaveTypeNormal, s.normalSlots, width, s.selectedTableSlot(manager.SaveTypeNormal), app.focus == focusContent && s.section == 0), "\n")...)
	lines = append(lines, "", t("Modded Saves"))
	lines = append(lines, strings.Split(s.renderSaveSlotTable(manager.SaveTypeModded, s.moddedSlots, width, s.selectedTableSlot(manager.SaveTypeModded), app.focus == focusContent && s.section == 0), "\n")...)
	return strings.Join(lines, "\n")
}

func (s *savesScreen) selectedTableSlot(saveType manager.SaveType) int {
	switch {
	case saveType == manager.SaveTypeNormal && s.listCursor >= 1 && s.listCursor <= 3:
		return s.listCursor
	case saveType == manager.SaveTypeModded && s.listCursor >= 4 && s.listCursor <= 6:
		return s.listCursor - 3
	default:
		return 0
	}
}

func (s *savesScreen) renderSaveSlotTable(saveType manager.SaveType, slots []manager.SaveSlotInfo, width int, selectedSlot int, focused bool) string {
	backupWidth := lipgloss.Width(t("%d backups", 0))
	for slot := 1; slot <= len(slots); slot++ {
		backupWidth = maxInt(backupWidth, lipgloss.Width(t("%d backups", s.backupCountForSlot(saveType, slot))))
	}
	indicatorWidth := 2
	slotWidth := 3
	statusWidth := maxInt(8, width-indicatorWidth-slotWidth-backupWidth)
	rows := make([]table.Row, 0, len(slots))
	for slot := 1; slot <= len(slots); slot++ {
		indicator := ""
		if selectedSlot == slot {
			indicator = ">"
		}
		rows = append(rows, table.Row{
			indicator,
			fmt.Sprintf("%d", slot),
			buildSlotStatus(slots[slot-1]),
			t("%d backups", s.backupCountForSlot(saveType, slot)),
		})
	}
	styles := table.DefaultStyles()
	styles.Header = lipgloss.NewStyle()
	styles.Cell = lipgloss.NewStyle()
	styles.Selected = lipgloss.NewStyle()
	if selectedSlot > 0 {
		styles.Selected = cursorStyle
		if focused {
			styles.Selected = focusStyle
		}
	}
	tbl := table.New(
		table.WithColumns([]table.Column{{Title: "", Width: indicatorWidth}, {Title: "", Width: slotWidth}, {Title: "", Width: statusWidth}, {Title: "", Width: backupWidth}}),
		table.WithRows(rows),
		table.WithWidth(width),
		table.WithHeight(len(rows)+1),
		table.WithStyles(styles),
	)
	if selectedSlot > 0 {
		tbl.SetCursor(selectedSlot - 1)
	}
	view := tbl.View()
	if cut := strings.Index(view, "\n"); cut >= 0 {
		view = view[cut+1:]
	}
	return view
}

func (s *savesScreen) renderRightPanel(app *appModel) string {
	selectedType, selectedSlot, info := s.selectedSlotInfo()
	lines := []string{
		t("Save Info"),
		t("Type: %s", formatSaveTypeName(selectedType)),
		t("Slot: %d", selectedSlot),
		t("Status: %s", buildSlotStatus(info)),
		t("Backups: %d", s.backupCountForSlot(selectedType, selectedSlot)),
		"",
		t("Save Actions"),
		renderActionLine(t("Copy Save"), false),
		renderActionLine(t("Backup Save"), false),
		"",
		strings.Join([]string{renderActionLine(t("Restore Backup"), false), renderActionLine(t("Delete Backup"), false)}, " "),
	}
	filtered := s.selectedBackups()
	for _, backupLine := range s.renderBackupLines(filtered, app.focus == focusContent && s.section == 1) {
		lines = append(lines, backupLine)
	}
	return strings.Join(lines, "\n")
}

func (s *savesScreen) renderBackupLines(backups []manager.BackupInfo, focused bool) []string {
	lines := []string{t("Backup List")}
	if len(backups) == 0 {
		return append(lines, mutedStyle.Render(t("No backups exist for the selected Steam profile yet.")))
	}
	for idx, backup := range backups {
		label := fmt.Sprintf("%s (%s)", backup.Name, backupTimestampText(backup.Name))
		lines = append(lines, renderSelectableLine(label, idx == s.backupCursor, focused))
	}
	return lines
}

func (s *savesScreen) switchSteamProfile(app *appModel, delta int) {
	if len(s.steamIDs) == 0 {
		return
	}
	current := 0
	for idx, steamID := range s.steamIDs {
		if steamID == app.state.SelectedSteamID() {
			current = idx
			break
		}
	}
	current = (current + delta + len(s.steamIDs)) % len(s.steamIDs)
	app.state.SetSelectedSteamID(s.steamIDs[current])
	app.logInfo("Switched active Steam profile to %s", s.steamIDs[current])
	s.refresh(app)
}

func (s *savesScreen) openCopyModal(app *appModel) {
	selectedType, selectedSlot, info := s.selectedSlotInfo()
	if !info.HasData {
		app.showInfo("Copy Save", t("Slot %s is empty, so there is nothing to copy.", formatSaveRef(selectedType, selectedSlot)))
		return
	}
	options := s.buildCopyOptions(selectedType, selectedSlot)
	app.showCopyTargetModal("Copy Options", options, func(model *appModel, option copyTargetOption, createBackup bool) {
		result, err := model.state.CopySave(selectedType, selectedSlot, option.SaveType, option.Slot, createBackup)
		if err != nil {
			model.showError("Copy save failed", err)
			return
		}
		model.logSuccess("Copied %d file(s): %s -> %s", result.CopiedFiles, formatSaveRef(selectedType, selectedSlot), formatSaveRef(option.SaveType, option.Slot))
		if result.BackupDir != "" {
			model.logInfo("Automatic backup created: %s", result.BackupDir)
		}
		if createBackup {
			model.logInfo("Created .before_copy files before overwriting target save data")
		}
		if result.CloudSynced {
			model.logInfo("Updated %d Steam cloud cache file(s)", result.CloudUpdated)
		}
		s.refresh(model)
	})
}

func (s *savesScreen) buildCopyOptions(sourceType manager.SaveType, sourceSlot int) []copyTargetOption {
	options := []copyTargetOption{{Header: true, Label: t("Vanilla Saves")}}
	for slot := 1; slot <= 3; slot++ {
		if sourceType == manager.SaveTypeNormal && slot == sourceSlot {
			continue
		}
		info := s.normalSlots[slot-1]
		options = append(options, copyTargetOption{Label: fmt.Sprintf("%d", slot), SaveType: manager.SaveTypeNormal, Slot: slot, Status: buildSlotStatus(info), HasData: info.HasData})
	}
	options = append(options, copyTargetOption{Header: true, Label: t("Modded Saves")})
	for slot := 1; slot <= 3; slot++ {
		if sourceType == manager.SaveTypeModded && slot == sourceSlot {
			continue
		}
		info := s.moddedSlots[slot-1]
		options = append(options, copyTargetOption{Label: fmt.Sprintf("%d", slot), SaveType: manager.SaveTypeModded, Slot: slot, Status: buildSlotStatus(info), HasData: info.HasData})
	}
	return options
}

func (s *savesScreen) backupSelected(app *appModel) {
	selectedType, selectedSlot, info := s.selectedSlotInfo()
	if !info.HasData {
		app.showInfo("Backup Save", t("Slot %s is empty, so there is nothing to back up.", formatSaveRef(selectedType, selectedSlot)))
		return
	}
	backupDir, err := app.state.BackupSave(selectedType, selectedSlot)
	if err != nil {
		app.showError("Backup failed", err)
		return
	}
	if backupDir == "" {
		app.logWarn("No save files were backed up from %s", formatSaveRef(selectedType, selectedSlot))
	} else {
		app.logSuccess("Created manual backup for %s at %s", formatSaveRef(selectedType, selectedSlot), backupDir)
	}
	s.refresh(app)
}

func (s *savesScreen) restoreSelected(app *appModel) {
	backups := s.selectedBackups()
	if len(backups) == 0 || s.backupCursor >= len(backups) {
		app.showInfo("Restore Backup", "Select a backup before restoring.")
		return
	}
	backup := backups[s.backupCursor]
	app.showConfirm("Confirm Restore", t("Restore backup %s to %s slot %d?", backup.Name, strings.ToLower(formatSaveTypeName(backup.Type)), backup.Slot), func(model *appModel) {
		result, err := model.state.RestoreBackup(backup, backup.Slot)
		if err != nil {
			model.showError("Restore failed", err)
			return
		}
		model.logSuccess("Restored backup %s into %s (%d file(s))", backup.Name, formatSaveRef(backup.Type, backup.Slot), result.CopiedFiles)
		if result.BackupDir != "" {
			model.logInfo("Backed up the current target first: %s", result.BackupDir)
		}
		if result.CloudSynced {
			model.logInfo("Updated %d Steam cloud cache file(s) after restore", result.CloudUpdated)
		}
		s.refresh(model)
	})
}

func (s *savesScreen) deleteSelected(app *appModel) {
	backups := s.selectedBackups()
	if len(backups) == 0 || s.backupCursor >= len(backups) {
		app.showInfo("Delete Backup", "Select a backup before deleting.")
		return
	}
	backup := backups[s.backupCursor]
	app.showConfirm("Delete Backup", t("Delete backup %s?", backup.Name), func(model *appModel) {
		if err := model.state.DeleteBackup(backup); err != nil {
			model.showError("Delete backup failed", err)
			return
		}
		model.logSuccess("Deleted backup %s", backup.Name)
		s.refresh(model)
	})
}

func (s *savesScreen) selectedSlotInfo() (manager.SaveType, int, manager.SaveSlotInfo) {
	cursor := s.listCursor
	if cursor == 0 {
		cursor = s.lastSlotCursor
	}
	if cursor >= 4 {
		slot := cursor - 3
		return manager.SaveTypeModded, slot, s.moddedSlots[slot-1]
	}
	slot := maxInt(1, cursor)
	return manager.SaveTypeNormal, slot, s.normalSlots[slot-1]
}

func (s *savesScreen) selectedBackups() []manager.BackupInfo {
	selectedType, selectedSlot, _ := s.selectedSlotInfo()
	filtered := make([]manager.BackupInfo, 0)
	for _, backup := range s.backups {
		if backup.Type == selectedType && backup.Slot == selectedSlot {
			filtered = append(filtered, backup)
		}
	}
	return filtered
}

func (s *savesScreen) backupCountForSlot(saveType manager.SaveType, slot int) int {
	count := 0
	for _, backup := range s.backups {
		if backup.Type == saveType && backup.Slot == slot {
			count++
		}
	}
	return count
}

func renderSelectableLine(text string, selected, focused bool) string {
	prefix := "  "
	style := lipgloss.NewStyle()
	if selected {
		prefix = "> "
		style = cursorStyle
		if focused {
			style = focusStyle
		}
	}
	return style.Render(prefix + text)
}

func backupTimestampText(name string) string {
	parts := strings.Split(name, "_")
	if len(parts) < 2 {
		return "-"
	}
	stamp := parts[len(parts)-2] + "_" + parts[len(parts)-1]
	if len(stamp) != len("20060102_150405") {
		return "-"
	}
	return stamp
}
