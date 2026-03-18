package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"slaymodgo/internal/manager"
)

type savesScreen struct {
	fieldCursor            int
	backupCursor           int
	section                int
	steamIDs               []string
	normalSlots            []manager.SaveSlotInfo
	moddedSlots            []manager.SaveSlotInfo
	backups                []manager.BackupInfo
	sourceType             manager.SaveType
	sourceSlot             int
	targetType             manager.SaveType
	targetSlot             int
	restoreSlot            int
	createBeforeCopyBackup bool
	err                    string
}

func (s *savesScreen) refresh(app *appModel) {
	if s.sourceSlot == 0 {
		s.sourceSlot = 1
	}
	if s.targetSlot == 0 {
		s.targetSlot = 1
	}
	if s.restoreSlot == 0 {
		s.restoreSlot = 1
	}
	if s.sourceType == "" {
		s.sourceType = manager.SaveTypeNormal
	}
	if s.targetType == "" {
		s.targetType = manager.SaveTypeModded
	}
	s.err = ""
	ids, err := app.state.ListSteamIDs()
	s.steamIDs = ids
	if err != nil {
		s.err = err.Error()
		return
	}
	if len(ids) == 0 {
		return
	}
	normalSlots, err := app.state.ListSaveSlots(manager.SaveTypeNormal)
	if err != nil {
		s.err = err.Error()
		return
	}
	moddedSlots, err := app.state.ListSaveSlots(manager.SaveTypeModded)
	if err != nil {
		s.err = err.Error()
		return
	}
	backups, err := app.state.ListBackups()
	if err != nil {
		s.err = err.Error()
		return
	}
	s.normalSlots = normalSlots
	s.moddedSlots = moddedSlots
	s.backups = backups
	if s.backupCursor >= len(s.backups) {
		s.backupCursor = maxInt(0, len(s.backups)-1)
	}
	if len(s.backups) > 0 && s.backups[s.backupCursor].Slot >= 1 && s.backups[s.backupCursor].Slot <= 3 {
		s.restoreSlot = s.backups[s.backupCursor].Slot
	}
}

func (s *savesScreen) handleKey(app *appModel, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "s":
		s.section = 1 - s.section
	case "up", "k":
		if s.section == 0 {
			if s.fieldCursor > 0 {
				s.fieldCursor--
			}
		} else if s.backupCursor > 0 {
			s.backupCursor--
			if len(s.backups) > 0 {
				s.restoreSlot = s.backups[s.backupCursor].Slot
			}
		}
	case "down", "j":
		if s.section == 0 {
			if s.fieldCursor < 6 {
				s.fieldCursor++
			}
		} else if s.backupCursor < len(s.backups)-1 {
			s.backupCursor++
			if len(s.backups) > 0 {
				s.restoreSlot = s.backups[s.backupCursor].Slot
			}
		}
	case "left", "h":
		s.adjustField(app, -1)
	case "right", "l":
		s.adjustField(app, 1)
	case "c":
		s.copySave(app)
	case "b":
		s.backupSource(app)
	case "enter", "r":
		if s.section == 1 {
			s.restoreSelected(app)
		}
	}
	return nil
}

func (s *savesScreen) adjustField(app *appModel, delta int) {
	if len(s.steamIDs) == 0 {
		return
	}
	switch s.fieldCursor {
	case 0:
		idx := 0
		for i, id := range s.steamIDs {
			if id == app.state.SelectedSteamID() {
				idx = i
				break
			}
		}
		idx = (idx + delta + len(s.steamIDs)) % len(s.steamIDs)
		app.state.SetSelectedSteamID(s.steamIDs[idx])
		app.logInfo("Switched active Steam profile to %s", s.steamIDs[idx])
		s.refresh(app)
	case 1:
		if s.sourceType == manager.SaveTypeNormal {
			s.sourceType = manager.SaveTypeModded
		} else {
			s.sourceType = manager.SaveTypeNormal
		}
	case 2:
		s.sourceSlot = rotateSlot(s.sourceSlot, delta)
	case 3:
		if s.targetType == manager.SaveTypeNormal {
			s.targetType = manager.SaveTypeModded
		} else {
			s.targetType = manager.SaveTypeNormal
		}
	case 4:
		s.targetSlot = rotateSlot(s.targetSlot, delta)
	case 5:
		s.restoreSlot = rotateSlot(s.restoreSlot, delta)
	case 6:
		s.createBeforeCopyBackup = !s.createBeforeCopyBackup
	}
}

func (s *savesScreen) copySave(app *appModel) {
	if s.sourceType == s.targetType && s.sourceSlot == s.targetSlot {
		app.showInfo("Copy Save", "Choose different source and target slots before copying.")
		return
	}
	sourceInfo, err := app.state.GetSaveSlotInfo(s.sourceType, s.sourceSlot)
	if err != nil {
		app.showError("Could not inspect source slot", err)
		return
	}
	if !sourceInfo.HasData {
		app.showInfo("Copy Save", fmt.Sprintf("Source slot %s is empty.", formatSaveRef(s.sourceType, s.sourceSlot)))
		return
	}
	targetInfo, err := app.state.GetSaveSlotInfo(s.targetType, s.targetSlot)
	if err != nil {
		app.showError("Could not inspect target slot", err)
		return
	}
	execute := func(model *appModel) {
		result, copyErr := model.state.CopySave(s.sourceType, s.sourceSlot, s.targetType, s.targetSlot, s.createBeforeCopyBackup)
		if copyErr != nil {
			model.showError("Copy save failed", copyErr)
			return
		}
		model.logSuccess("Copied %d file(s): %s -> %s", result.CopiedFiles, formatSaveRef(s.sourceType, s.sourceSlot), formatSaveRef(s.targetType, s.targetSlot))
		if result.BackupDir != "" {
			model.logInfo("Automatic backup created: %s", result.BackupDir)
		}
		if result.CloudSynced {
			model.logInfo("Updated %d Steam cloud cache file(s)", result.CloudUpdated)
		}
		s.refresh(model)
	}
	if targetInfo.HasData {
		app.showConfirm("Confirm Save Copy", fmt.Sprintf("Target slot %s already has data. The manager will back it up first and then overwrite it. Continue?", formatSaveRef(s.targetType, s.targetSlot)), execute)
		return
	}
	execute(app)
}

func (s *savesScreen) backupSource(app *appModel) {
	info, err := app.state.GetSaveSlotInfo(s.sourceType, s.sourceSlot)
	if err != nil {
		app.showError("Could not inspect source slot", err)
		return
	}
	if !info.HasData {
		app.showInfo("Backup Save", fmt.Sprintf("Slot %s is empty, so there is nothing to back up.", formatSaveRef(s.sourceType, s.sourceSlot)))
		return
	}
	backupDir, err := app.state.BackupSave(s.sourceType, s.sourceSlot)
	if err != nil {
		app.showError("Backup failed", err)
		return
	}
	if backupDir == "" {
		app.logWarn("No save files were backed up from %s", formatSaveRef(s.sourceType, s.sourceSlot))
	} else {
		app.logSuccess("Created manual backup for %s at %s", formatSaveRef(s.sourceType, s.sourceSlot), backupDir)
	}
	s.refresh(app)
}

func (s *savesScreen) restoreSelected(app *appModel) {
	if len(s.backups) == 0 || s.backupCursor >= len(s.backups) {
		app.showInfo("Restore Backup", "Select a backup before restoring.")
		return
	}
	backup := s.backups[s.backupCursor]
	targetInfo, err := app.state.GetSaveSlotInfo(backup.Type, s.restoreSlot)
	if err != nil {
		app.showError("Could not inspect restore target", err)
		return
	}
	message := fmt.Sprintf("Restore backup %s to %s slot %d?", backup.Name, strings.ToLower(formatSaveTypeName(backup.Type)), s.restoreSlot)
	if targetInfo.HasData {
		message += "\n\nThe current target slot has data and will be backed up automatically before restore."
	}
	app.showConfirm("Confirm Restore", message, func(model *appModel) {
		result, restoreErr := model.state.RestoreBackup(backup, s.restoreSlot)
		if restoreErr != nil {
			model.showError("Restore failed", restoreErr)
			return
		}
		model.logSuccess("Restored backup %s into %s (%d file(s))", backup.Name, formatSaveRef(backup.Type, s.restoreSlot), result.CopiedFiles)
		if result.BackupDir != "" {
			model.logInfo("Backed up the current target first: %s", result.BackupDir)
		}
		if result.CloudSynced {
			model.logInfo("Updated %d Steam cloud cache file(s) after restore", result.CloudUpdated)
		}
		s.refresh(model)
	})
}

func (s *savesScreen) view(app *appModel, width, height int) string {
	if s.err != "" {
		return renderFlatColumn("Status", "Save management error\n\n"+s.err, width, height)
	}
	if len(s.steamIDs) == 0 {
		return renderFlatColumn("Status", "No Steam save profiles were found under:\n\n"+app.manager.SaveRoot+"\n\nLaunch the game once to create the save folders, then refresh this screen.", width, height)
	}
	fields := []string{
		fmt.Sprintf("Steam ID: %s", app.state.SelectedSteamID()),
		fmt.Sprintf("Source Type: %s", formatSaveTypeName(s.sourceType)),
		fmt.Sprintf("Source Slot: %d", s.sourceSlot),
		fmt.Sprintf("Target Type: %s", formatSaveTypeName(s.targetType)),
		fmt.Sprintf("Target Slot: %d", s.targetSlot),
		fmt.Sprintf("Restore Slot: %d", s.restoreSlot),
		fmt.Sprintf("Create .before_copy: %t", s.createBeforeCopyBackup),
	}
	fieldList := renderList(fields, s.fieldCursor, app.focus == focusContent && s.section == 0)

	var slots strings.Builder
	slots.WriteString("Save Slots\n\n")
	slots.WriteString("Slot  Vanilla               Modded\n")
	for idx := 0; idx < 3; idx++ {
		slots.WriteString(fmt.Sprintf("%s   %-20s %s\n", formatSaveRef(manager.SaveTypeNormal, idx+1), buildSlotStatus(s.normalSlots[idx]), buildSlotStatus(s.moddedSlots[idx])))
	}

	backupItems := make([]string, 0, len(s.backups))
	for _, backup := range s.backups {
		backupItems = append(backupItems, fmt.Sprintf("%s slot %d (%d files)", formatSaveTypeName(backup.Type), backup.Slot, backup.FileCount))
	}
	backupList := renderList(backupItems, s.backupCursor, app.focus == focusContent && s.section == 1)
	backupDetail := "No backups exist for the selected Steam profile yet."
	if len(s.backups) > 0 && s.backupCursor < len(s.backups) {
		backupDetail = renderBackupDetail(s.backups[s.backupCursor])
	}

	leftWidth, rightWidth := splitContentWidths(width, 30, 24)
	leftBody := strings.Join([]string{fieldList, "", slots.String()}, "\n")
	rightBody := strings.Join([]string{"Backup List", "", backupList, "", "Selected Backup", "", backupDetail}, "\n")
	left := renderFlatColumn("Copy and Restore", leftBody, leftWidth, height)
	right := renderFlatColumn("Backups", rightBody, rightWidth, height)
	return joinFlatColumns(left, right, leftWidth, rightWidth)
}

func (s *savesScreen) help() string {
	return "Saves: s switch section | up/down move | left/right change | c copy | b backup | enter/r restore"
}

func rotateSlot(value, delta int) int {
	value += delta
	if value < 1 {
		return 3
	}
	if value > 3 {
		return 1
	}
	return value
}
