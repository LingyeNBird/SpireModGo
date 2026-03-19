package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"slaymodgo/internal/manager"
)

type modsTab int

const (
	modsTabAvailable modsTab = iota
	modsTabInstalled
)

type modsScreen struct {
	tab               modsTab
	available         []manager.ModPackage
	installed         []manager.InstalledMod
	availableSelected map[string]bool
	installedSelected map[string]bool
	availableCursor   int
	installedCursor   int
	availableErr      string
	installedErr      string
}

func (s *modsScreen) refresh(app *appModel) {
	available, availableErr := app.state.ListAvailableMods()
	s.available = available
	s.availableErr = ""
	if availableErr != nil {
		s.availableErr = app.localizeError(availableErr)
	}
	installed, installedErr := app.state.ListInstalledMods()
	s.installed = installed
	s.installedErr = ""
	if installedErr != nil {
		s.installedErr = app.localizeError(installedErr)
	}
	if s.availableSelected == nil {
		s.availableSelected = map[string]bool{}
	}
	if s.installedSelected == nil {
		s.installedSelected = map[string]bool{}
	}
	if s.availableCursor >= len(s.available) {
		s.availableCursor = maxInt(0, len(s.available)-1)
	}
	if s.installedCursor >= len(s.installed) {
		s.installedCursor = maxInt(0, len(s.installed)-1)
	}
	if s.tab != modsTabInstalled {
		s.tab = modsTabAvailable
	}
}

func (s *modsScreen) handleKey(app *appModel, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "left", "h":
		s.tab = modsTabAvailable
	case "right", "l":
		s.tab = modsTabInstalled
	case "tab":
		if s.tab == modsTabAvailable {
			s.tab = modsTabInstalled
		} else {
			s.tab = modsTabAvailable
		}
	case "up", "k":
		if s.tab == modsTabAvailable && s.availableCursor > 0 {
			s.availableCursor--
		}
		if s.tab == modsTabInstalled && s.installedCursor > 0 {
			s.installedCursor--
		}
	case "down", "j":
		if s.tab == modsTabAvailable && s.availableCursor < len(s.available)-1 {
			s.availableCursor++
		}
		if s.tab == modsTabInstalled && s.installedCursor < len(s.installed)-1 {
			s.installedCursor++
		}
	case " ", "enter":
		s.toggleCurrent()
	case "a":
		s.selectAllCurrent()
	case "c":
		s.clearCurrentSelection()
	case "i":
		if s.tab == modsTabAvailable {
			s.installSelected(app)
		}
	case "d":
		if s.tab == modsTabInstalled {
			s.uninstallSelected(app)
		}
	case "f":
		s.repairCurrent(app)
	}
	return nil
}

func (s *modsScreen) handleMouse(app *appModel, msg tea.MouseMsg, x, y, width, height int) tea.Cmd {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil
	}
	leftWidth := maxInt(1, (width-3)/2)
	layout := newSplitBodyLayout(width, height, leftWidth)
	if !layout.leftBody.contains(x, y) {
		if layout.rightBody.contains(x, y) {
			localX := x - layout.rightBody.x
			localY := y - layout.rightBody.y
			if localY == s.repairButtonRow(layout.rightBody.width) && s.currentNeedsRepair() {
				if inlineButtonIndexAt([]string{t("Repair Mod")}, localX) == 0 {
					s.repairCurrent(app)
				}
			}
		}
		return nil
	}
	localX := x - layout.leftBody.x
	localY := y - layout.leftBody.y
	switch {
	case localY == 0:
		switch s.tabIndexAt(localX) {
		case 0:
			s.tab = modsTabAvailable
		case 1:
			s.tab = modsTabInstalled
		}
	case localY == 2:
		switch s.actionIndexAt(localX) {
		case 0:
			s.selectAllCurrent()
		case 1:
			s.clearCurrentSelection()
		case 2:
			if s.tab == modsTabAvailable {
				s.installSelected(app)
			} else {
				s.uninstallSelected(app)
			}
		}
	case localY >= 3:
		idx := localY - 3
		if s.tab == modsTabAvailable && idx >= 0 && idx < len(s.available) {
			s.availableCursor = idx
			key := s.available[idx].InstallName
			s.availableSelected[key] = !s.availableSelected[key]
		}
		if s.tab == modsTabInstalled && idx >= 0 && idx < len(s.installed) {
			s.installedCursor = idx
			key := s.installed[idx].DirName
			s.installedSelected[key] = !s.installedSelected[key]
		}
	}
	return nil
}

func (s *modsScreen) currentAvailableSelection() []manager.ModPackage {
	chosen := make([]manager.ModPackage, 0)
	for _, mod := range s.available {
		if s.availableSelected[mod.InstallName] {
			chosen = append(chosen, mod)
		}
	}
	return chosen
}

func (s *modsScreen) currentInstalledSelection() []string {
	chosen := make([]string, 0)
	for _, mod := range s.installed {
		if s.installedSelected[mod.DirName] {
			chosen = append(chosen, mod.DirName)
		}
	}
	return chosen
}

func (s *modsScreen) installSelected(app *appModel) {
	items := s.currentAvailableSelection()
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
	s.availableSelected = map[string]bool{}
	s.refresh(app)
	app.offerPostInstallSaveCopy()
}

func (s *modsScreen) uninstallSelected(app *appModel) {
	names := s.currentInstalledSelection()
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
		s.installedSelected = map[string]bool{}
		s.refresh(model)
	})
}

func (s *modsScreen) toggleCurrent() {
	if s.tab == modsTabAvailable {
		if len(s.available) == 0 {
			return
		}
		key := s.available[s.availableCursor].InstallName
		s.availableSelected[key] = !s.availableSelected[key]
		return
	}
	if len(s.installed) == 0 {
		return
	}
	key := s.installed[s.installedCursor].DirName
	s.installedSelected[key] = !s.installedSelected[key]
}

func (s *modsScreen) selectAllCurrent() {
	if s.tab == modsTabAvailable {
		for _, mod := range s.available {
			s.availableSelected[mod.InstallName] = true
		}
		return
	}
	for _, mod := range s.installed {
		s.installedSelected[mod.DirName] = true
	}
}

func (s *modsScreen) clearCurrentSelection() {
	if s.tab == modsTabAvailable {
		s.availableSelected = map[string]bool{}
		return
	}
	s.installedSelected = map[string]bool{}
}

func (s *modsScreen) view(app *appModel, width, height int) string {
	tabLine := s.renderTabs()
	actionLine := s.renderActions()
	leftWidth := maxInt(1, (width-3)/2)
	layout := newSplitBodyLayout(width, height, leftWidth)
	if s.tab == modsTabAvailable {
		if s.availableErr != "" {
			return renderSplitBody(t("Mod Management"), strings.Join([]string{tabLine, "", s.availableErr}, "\n"), t("Mod Details"), t("Failed to load available packages:\n\n%s", s.availableErr), width, height, leftWidth)
		}
		items := make([]string, 0, len(s.available))
		for idx, mod := range s.available {
			items = append(items, renderModListEntry(mod, s.availableSelected[mod.InstallName], idx == s.availableCursor, app.focus == focusContent))
		}
		detail := t("No packages were found in the bundled Mods directory.")
		if len(s.available) > 0 {
			detail = wrapBodyText(renderAvailableModDetail(s.available[s.availableCursor]), layout.rightBody.width)
		}
		if s.currentNeedsRepair() {
			detail = strings.Join(append(s.renderRepairHeader(layout.rightBody.width), detail), "\n")
		}
		leftBody := strings.Join([]string{tabLine, "", actionLine, renderModsList(items)}, "\n")
		return renderSplitBody(t("Mod Management"), leftBody, t("Mod Details"), detail, width, height, leftWidth)
	}
	detail := t("No installed mods were found in the game's mods directory.")
	if s.installedErr != "" {
		detail = wrapBodyText(s.installedErr+"\n\n"+t("Open Settings to configure the game path before uninstalling."), layout.rightBody.width)
	}
	items := make([]string, 0, len(s.installed))
	for idx, mod := range s.installed {
		items = append(items, renderInstalledModListEntry(mod, s.installedSelected[mod.DirName], idx == s.installedCursor, app.focus == focusContent))
	}
	if s.installedErr == "" && len(s.installed) > 0 {
		detail = wrapBodyText(renderInstalledModDetail(s.installed[s.installedCursor]), layout.rightBody.width)
	}
	if s.currentNeedsRepair() {
		detail = strings.Join(append(s.renderRepairHeader(layout.rightBody.width), detail), "\n")
	}
	leftBody := strings.Join([]string{tabLine, "", actionLine, renderModsList(items)}, "\n")
	return renderSplitBody(t("Mod Management"), leftBody, t("Mod Details"), detail, width, height, leftWidth)
}

func (s *modsScreen) renderRepairHeader(width int) []string {
	warningLines := strings.Split(wrapBodyText(t("This mod format seems incompatible with the new Slay the Spire version. Click to repair."), width), "\n")
	lines := make([]string, 0, len(warningLines)+2)
	for _, line := range warningLines {
		lines = append(lines, errorStyle.Render(line))
	}
	lines = append(lines, renderInlineButton(t("Repair Mod"), false, false), "")
	return lines
}

func (s *modsScreen) repairButtonRow(width int) int {
	if !s.currentNeedsRepair() {
		return -1
	}
	return len(strings.Split(wrapBodyText(t("This mod format seems incompatible with the new Slay the Spire version. Click to repair."), width), "\n"))
}

func (s *modsScreen) currentNeedsRepair() bool {
	if s.tab == modsTabAvailable {
		return len(s.available) > 0 && s.available[s.availableCursor].NeedsRepair
	}
	return len(s.installed) > 0 && s.installed[s.installedCursor].NeedsRepair
}

func (s *modsScreen) repairCurrent(app *appModel) {
	if !s.currentNeedsRepair() {
		return
	}
	if s.tab == modsTabAvailable {
		mod := s.available[s.availableCursor]
		result, err := app.state.RepairAvailableMod(mod.DirName)
		if err != nil {
			app.showError("Repair mod failed", err)
			return
		}
		app.logSuccess("Repaired mod config for %s", mod.Label)
		app.logInfo("Generated config: %s", result.ConfigPath)
		if result.RemovedLegacyManifest {
			app.logInfo("Removed legacy mod_manifest.json")
		}
		s.refresh(app)
		return
	}
	mod := s.installed[s.installedCursor]
	result, err := app.state.RepairInstalledMod(mod.DirName)
	if err != nil {
		app.showError("Repair mod failed", err)
		return
	}
	app.logSuccess("Repaired mod config for %s", mod.Label)
	app.logInfo("Generated config: %s", result.ConfigPath)
	if result.RemovedLegacyManifest {
		app.logInfo("Removed legacy mod_manifest.json")
	}
	s.refresh(app)
}

func (s *modsScreen) actionLabels() []string {
	labels := []string{t("Select All"), t("Clear Selection")}
	if s.tab == modsTabAvailable {
		return append(labels, t("Install"))
	}
	return append(labels, t("Uninstall"))
}

func (s *modsScreen) actionIndexAt(localX int) int {
	return inlineButtonIndexAt(s.actionLabels(), localX)
}

func (s *modsScreen) renderTabs() string {
	labels := s.tabLabels()
	rendered := make([]string, 0, len(labels))
	for idx, label := range labels {
		rendered = append(rendered, renderModsTabLabel(label, idx == int(s.tab)))
	}
	return strings.Join(rendered, " ")
}

func (s *modsScreen) tabLabels() []string {
	return []string{t("Not Installed [%d]", len(s.available)), t("Installed [%d]", len(s.installed))}
}

func (s *modsScreen) tabIndexAt(localX int) int {
	labels := s.tabLabels()
	offset := 0
	for idx, label := range labels {
		width := lipgloss.Width(formatModsTabLabel(label, idx == int(s.tab)))
		if localX >= offset && localX < offset+width {
			return idx
		}
		offset += width
		if idx < len(labels)-1 {
			offset++
		}
	}
	return -1
}

func formatModsTabLabel(label string, active bool) string {
	if active {
		return "> " + label + " <"
	}
	return "  " + label + "  "
}

func renderModsTabLabel(label string, active bool) string {
	text := formatModsTabLabel(label, active)
	if active {
		return navActiveStyle.Render(text)
	}
	return navItemStyle.Render(text)
}

func renderModsList(items []string) string {
	if len(items) == 0 {
		return mutedStyle.Render(t("(empty)"))
	}
	return strings.Join(items, "\n")
}

func (s *modsScreen) renderActions() string {
	return renderInlineButtonGroup(s.actionLabels(), -1, false)
}

func (s *modsScreen) help() helpSection {
	return helpSection{
		Title: t("Mods:"),
		Items: []helpItem{
			{Action: t("left/right"), Description: t("switch tab")},
			{Action: t("Click"), Description: t("tabs and actions")},
			{Action: t("up/down"), Description: t("move")},
			{Action: t("space"), Description: t("toggle")},
			{Action: t("i"), Description: t("install")},
			{Action: t("d"), Description: t("uninstall")},
		},
	}
}
