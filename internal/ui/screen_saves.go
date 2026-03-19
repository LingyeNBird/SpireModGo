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

func (s *savesScreen) handleMouse(app *appModel, msg tea.MouseMsg, x, y, width, height int) tea.Cmd {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil
	}
	leftWidth, _ := splitContentWidths(width, 30, 24)
	layout := newSplitBodyLayout(width, height, leftWidth)
	if layout.leftBody.contains(x, y) {
		localX, localY := x-layout.leftBody.x, y-layout.leftBody.y
		s.section = 0
		switch {
		case localY >= 0 && localY <= 6:
			s.fieldCursor = localY
			if s.fieldCursor == 6 {
				s.adjustField(app, 1)
				return nil
			}
			delta := 1
			if localX < layout.leftBody.width/2 {
				delta = -1
			}
			s.adjustField(app, delta)
		case localY == 8:
			s.copySave(app)
		case localY == 9:
			s.backupSource(app)
		}
		return nil
	}
	if layout.rightBody.contains(x, y) {
		localY := y - layout.rightBody.y
		s.section = 1
		switch {
		case localY == 0:
			s.restoreSelected(app)
		case localY >= 2:
			idx := localY - 2
			if idx >= 0 && idx < len(s.backups) {
				s.backupCursor = idx
				s.restoreSlot = s.backups[idx].Slot
			}
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
		app.showInfo("Copy Save", t("Source slot %s is empty.", formatSaveRef(s.sourceType, s.sourceSlot)))
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
		app.showConfirm("Confirm Save Copy", t("Target slot %s already has data. The manager will back it up first and then overwrite it. Continue?", formatSaveRef(s.targetType, s.targetSlot)), execute)
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
		app.showInfo("Backup Save", t("Slot %s is empty, so there is nothing to back up.", formatSaveRef(s.sourceType, s.sourceSlot)))
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
	message := t("Restore backup %s to %s slot %d?", backup.Name, strings.ToLower(formatSaveTypeName(backup.Type)), s.restoreSlot)
	if targetInfo.HasData {
		message += "\n\n" + t("The current target slot has data and will be backed up automatically before restore.")
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
		return t("Save management error\n\n%s", s.err)
	}
	if len(s.steamIDs) == 0 {
		return t("No Steam save profiles were found under:\n\n%s\n\nLaunch the game once to create the save folders, then refresh this screen.", app.manager.SaveRoot)
	}
	fields := []string{
		renderValueControl(t("Steam ID"), app.state.SelectedSteamID()),
		renderValueControl(t("Source Type"), formatSaveTypeName(s.sourceType)),
		renderValueControl(t("Source Slot"), fmt.Sprintf("%d", s.sourceSlot)),
		renderValueControl(t("Target Type"), formatSaveTypeName(s.targetType)),
		renderValueControl(t("Target Slot"), fmt.Sprintf("%d", s.targetSlot)),
		renderValueControl(t("Restore Slot"), fmt.Sprintf("%d", s.restoreSlot)),
		renderValueControl(t("Create .before_copy"), fmt.Sprintf("%t", s.createBeforeCopyBackup)),
	}
	fieldList := renderList(fields, s.fieldCursor, app.focus == focusContent && s.section == 0)

	var slots strings.Builder
	slots.WriteString(t("Save Slots") + "\n\n")
	slots.WriteString(t("Slot  Vanilla               Modded") + "\n")
	for idx := 0; idx < 3; idx++ {
		slots.WriteString(fmt.Sprintf("%s   %-20s %s\n", formatSaveRef(manager.SaveTypeNormal, idx+1), buildSlotStatus(s.normalSlots[idx]), buildSlotStatus(s.moddedSlots[idx])))
	}

	backupItems := make([]string, 0, len(s.backups))
	for _, backup := range s.backups {
		backupItems = append(backupItems, t("%s slot %d (%d files)", formatSaveTypeName(backup.Type), backup.Slot, backup.FileCount))
	}
	backupList := renderList(backupItems, s.backupCursor, app.focus == focusContent && s.section == 1)
	backupDetail := t("No backups exist for the selected Steam profile yet.")
	if len(s.backups) > 0 && s.backupCursor < len(s.backups) {
		backupDetail = renderBackupDetail(s.backups[s.backupCursor])
	}

	leftWidth, _ := splitContentWidths(width, 30, 24)
	leftBody := strings.Join([]string{fieldList, "", renderActionLine(t("Copy Save"), false), renderActionLine(t("Backup Save"), false), "", slots.String()}, "\n")
	rightBody := strings.Join([]string{renderActionLine(t("Restore Backup"), false), "", backupList, "", t("Selected Backup"), "", backupDetail}, "\n")
	return renderSplitBody(t("Copy and Restore"), leftBody, t("Backups"), rightBody, width, height, leftWidth)
}

func (s *savesScreen) help() string {
	return t("Saves: click fields to change | click backup rows and action buttons | left/right change | c copy | b backup | enter/r restore")
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
