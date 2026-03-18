package ui

import (
	"fmt"
	"strings"

	"slaymodgo/internal/manager"
)

func (m *appModel) offerPostInstallSaveCopy() {
	ids, err := m.state.ListSteamIDs()
	if err != nil {
		m.logWarn("Could not inspect save profiles after install: %v", err)
		return
	}
	if len(ids) == 0 {
		m.logInfo("Install finished. No Steam save profiles were found, so there is no save copy to suggest yet.")
		return
	}
	if m.state.SelectedSteamID() == "" {
		m.state.SetSelectedSteamID(ids[0])
	}

	normalSlots, err := m.state.ListSaveSlots(manager.SaveTypeNormal)
	if err != nil {
		m.logWarn("Could not inspect vanilla saves after install: %v", err)
		return
	}
	moddedSlots, err := m.state.ListSaveSlots(manager.SaveTypeModded)
	if err != nil {
		m.logWarn("Could not inspect modded saves after install: %v", err)
		return
	}

	hasNormal := false
	suggestedSource := 1
	suggestedTarget := 1
	newestNormalIndex := -1
	newestModdedIndex := -1
	for idx := 0; idx < 3; idx++ {
		if normalSlots[idx].HasData {
			if !hasNormal {
				suggestedSource = idx + 1
				hasNormal = true
			}
			if newestNormalIndex == -1 || normalSlots[idx].LastModified.After(normalSlots[newestNormalIndex].LastModified) {
				newestNormalIndex = idx
			}
		}
		if moddedSlots[idx].HasData && (newestModdedIndex == -1 || moddedSlots[idx].LastModified.After(moddedSlots[newestModdedIndex].LastModified)) {
			newestModdedIndex = idx
		}
	}
	if !hasNormal {
		m.showInfo("Install Complete", "Mods were installed, but no vanilla save slots were found to copy into modded mode yet.")
		return
	}
	for idx := 0; idx < 3; idx++ {
		if !moddedSlots[idx].HasData {
			suggestedTarget = idx + 1
			break
		}
	}

	moddedNewer := newestNormalIndex >= 0 && newestModdedIndex >= 0 && moddedSlots[newestModdedIndex].LastModified.After(normalSlots[newestNormalIndex].LastModified)

	var summary strings.Builder
	summary.WriteString(t("Vanilla saves:\n"))
	for _, slot := range normalSlots {
		summary.WriteString(fmt.Sprintf("- %s: %s\n", formatSaveRef(manager.SaveTypeNormal, slot.Slot), slotSummaryText(slot)))
	}
	summary.WriteString("\n" + t("Modded saves:\n"))
	for _, slot := range moddedSlots {
		summary.WriteString(fmt.Sprintf("- %s: %s\n", formatSaveRef(manager.SaveTypeModded, slot.Slot), slotSummaryText(slot)))
	}
	summary.WriteString("\n" + t("Suggested copy: %s -> %s", formatSaveRef(manager.SaveTypeNormal, suggestedSource), formatSaveRef(manager.SaveTypeModded, suggestedTarget)))
	if moddedNewer {
		summary.WriteString("\n\n" + t("Warning: your newest modded save is newer than your newest vanilla save. Copying now can overwrite newer modded progress."))
	}

	m.showConfirm("Copy Vanilla Save To Modded?", summary.String(), func(app *appModel) {
		result, copyErr := app.state.CopySave(manager.SaveTypeNormal, suggestedSource, manager.SaveTypeModded, suggestedTarget, true)
		if copyErr != nil {
			app.showError("Suggested save copy failed", copyErr)
			return
		}
		app.logSuccess("Copied suggested save %s -> %s (%d file(s))", formatSaveRef(manager.SaveTypeNormal, suggestedSource), formatSaveRef(manager.SaveTypeModded, suggestedTarget), result.CopiedFiles)
		if result.BackupDir != "" {
			app.logInfo("Automatic backup created before copy: %s", result.BackupDir)
		}
		if result.CloudSynced {
			app.logInfo("Updated %d Steam cloud cache file(s) after suggested copy", result.CloudUpdated)
		}
		if app.current == screenSaves {
			app.refreshCurrentScreen()
		}
	})
}

func slotSummaryText(slot manager.SaveSlotInfo) string {
	if !slot.HasData {
		return t("empty")
	}
	text := formatTimestamp(slot.LastModified)
	if slot.HasCurrentRun {
		text += " | " + t("current run")
	}
	return text
}
