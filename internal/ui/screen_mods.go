package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

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
		return nil
	}
	localX := x - layout.leftBody.x
	localY := y - layout.leftBody.y
	switch {
	case localY == 0:
		if localX < layout.leftBody.width/2 {
			s.tab = modsTabAvailable
		} else {
			s.tab = modsTabInstalled
		}
	case localY == 2:
		if localX < layout.leftBody.width/3 {
			s.selectAllCurrent()
		} else if localX < (layout.leftBody.width/3)*2 {
			s.clearCurrentSelection()
		} else if s.tab == modsTabAvailable {
			s.installSelected(app)
		} else {
			s.uninstallSelected(app)
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
	if s.tab == modsTabAvailable {
		if s.availableErr != "" {
			return renderSplitBody(t("Mod Management"), strings.Join([]string{tabLine, "", s.availableErr}, "\n"), t("Mod Details"), t("Failed to load available packages:\n\n%s", s.availableErr), width, height, leftWidth)
		}
		items := make([]string, 0, len(s.available))
		for _, mod := range s.available {
			items = append(items, renderModListLabel(mod, s.availableSelected[mod.InstallName]))
		}
		detail := t("No packages were found in the bundled Mods directory.")
		if len(s.available) > 0 {
			detail = renderAvailableModDetail(s.available[s.availableCursor])
		}
		leftBody := strings.Join([]string{tabLine, "", actionLine, renderList(items, s.availableCursor, app.focus == focusContent)}, "\n")
		return renderSplitBody(t("Mod Management"), leftBody, t("Mod Details"), detail, width, height, leftWidth)
	}
	detail := t("No installed mods were found in the game's mods directory.")
	if s.installedErr != "" {
		detail = s.installedErr + "\n\n" + t("Open Settings to configure the game path before uninstalling.")
	}
	items := make([]string, 0, len(s.installed))
	for _, mod := range s.installed {
		items = append(items, renderInstalledModListLabel(mod, s.installedSelected[mod.DirName]))
	}
	if s.installedErr == "" && len(s.installed) > 0 {
		detail = renderInstalledModDetail(s.installed[s.installedCursor])
	}
	leftBody := strings.Join([]string{tabLine, "", actionLine, renderList(items, s.installedCursor, app.focus == focusContent)}, "\n")
	return renderSplitBody(t("Mod Management"), leftBody, t("Mod Details"), detail, width, height, leftWidth)
}

func (s *modsScreen) renderTabs() string {
	available := t("Not Installed [%d]", len(s.available))
	installed := t("Installed [%d]", len(s.installed))
	if s.tab == modsTabAvailable {
		available = ">" + available + "<"
	} else {
		installed = ">" + installed + "<"
	}
	return available + "  " + installed
}

func (s *modsScreen) renderActions() string {
	if s.tab == modsTabAvailable {
		return strings.Join([]string{t("Select All"), t("Clear Selection"), t("Install")}, "  ")
	}
	return strings.Join([]string{t("Select All"), t("Clear Selection"), t("Uninstall")}, "  ")
}

func (s *modsScreen) help() string {
	return t("Mods: left/right switch tab | click tabs and actions | up/down move | space toggle | i install | d uninstall")
}
