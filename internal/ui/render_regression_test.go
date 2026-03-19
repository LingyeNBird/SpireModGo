package ui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"slaymodgo/internal/manager"
)

func loadTestLocalizer(tt *testing.T) {
	tt.Helper()
	if err := loadLocalizer(filepath.Clean(filepath.Join("..", ".."))); err != nil {
		tt.Fatalf("load localizer: %v", err)
	}
}

func TestWrapBodyTextWrapsWithoutDroppingContent(tt *testing.T) {
	text := "当前路径与状态当前路径与状态"
	got := wrapBodyText(text, 8)
	if !strings.Contains(got, "\n") {
		tt.Fatalf("expected wrapped text to contain a newline, got %q", got)
	}
	if strings.ReplaceAll(got, "\n", "") != text {
		tt.Fatalf("expected wrapped text to preserve content, got %q", got)
	}
}

func TestOverlayModalPreservesBackgroundOutsideModal(tt *testing.T) {
	base := strings.Join([]string{"0123456789", "abcdefghij"}, "\n")
	overlay := strings.Join([]string{"   ╭─╮    ", "   │X│    "}, "\n")
	got := overlayModal(base, overlay, 10, 2)
	lines := strings.Split(got, "\n")
	if lines[0] != "012╭─╮6789" {
		tt.Fatalf("expected top line to preserve background around modal, got %q", lines[0])
	}
	if lines[1] != "abc│X│ghij" {
		tt.Fatalf("expected body line to preserve background around modal, got %q", lines[1])
	}
}

func TestRenderSaveSlotTableAlignsBackupCounts(tt *testing.T) {
	loadTestLocalizer(tt)
	screen := savesScreen{backups: make([]manager.BackupInfo, 0, 17)}
	for range 12 {
		screen.backups = append(screen.backups, manager.BackupInfo{Type: manager.SaveTypeNormal, Slot: 1})
	}
	for range 5 {
		screen.backups = append(screen.backups, manager.BackupInfo{Type: manager.SaveTypeNormal, Slot: 3})
	}
	slots := []manager.SaveSlotInfo{
		{HasData: true, LastModified: time.Date(2026, 3, 18, 23, 42, 0, 0, time.UTC)},
		{HasData: false},
		{HasData: false},
	}
	view := ansi.Strip(screen.renderSaveSlotTable(manager.SaveTypeNormal, slots, 36, 1, true))
	lines := strings.Split(view, "\n")
	if len(lines) != 3 {
		tt.Fatalf("expected three save rows, got %d", len(lines))
	}
	labels := []string{t("%d backups", 12), t("%d backups", 0), t("%d backups", 5)}
	positions := make([]int, len(lines))
	for idx, line := range lines {
		byteIndex := strings.Index(line, labels[idx])
		if byteIndex < 0 {
			tt.Fatalf("expected backup label %q in line %q", labels[idx], line)
		}
		positions[idx] = lipgloss.Width(line[:byteIndex])
	}
	if positions[0] != positions[1] || positions[1] != positions[2] {
		tt.Fatalf("expected aligned backup columns, got positions %v in %q", positions, lines)
	}
	if !strings.HasPrefix(lines[0], ">") {
		tt.Fatalf("expected selected row to keep a visible cursor, got %q", lines[0])
	}
}

func TestModsPrimaryActionIndexAtUsesRenderedLabelWidths(tt *testing.T) {
	loadTestLocalizer(tt)
	screen := modsScreen{tab: modsTabAvailable}
	labels := screen.primaryActionLabels()
	firstWidth := lipgloss.Width(formatButtonLabel(labels[0]))
	secondWidth := lipgloss.Width(formatButtonLabel(labels[1]))
	thirdWidth := lipgloss.Width(formatButtonLabel(labels[2]))
	if got := screen.actionIndexAt(0, firstWidth/2); got != 0 {
		tt.Fatalf("expected select-all hit to map to index 0, got %d", got)
	}
	cancelX := firstWidth + 1 + secondWidth/2
	if got := screen.actionIndexAt(0, cancelX); got != 1 {
		tt.Fatalf("expected cancel hit to map to index 1, got %d", got)
	}
	installX := firstWidth + 1 + secondWidth + 1 + thirdWidth/2
	if got := screen.actionIndexAt(0, installX); got != 2 {
		tt.Fatalf("expected install hit to map to index 2, got %d", got)
	}
	if got := screen.actionIndexAt(0, firstWidth); got != -1 {
		tt.Fatalf("expected separator gap to be non-clickable, got %d", got)
	}
}

func TestModsSecondaryActionIndexAtUsesRenderedLabelWidths(tt *testing.T) {
	loadTestLocalizer(tt)
	screen := modsScreen{tab: modsTabAvailable}
	labels := screen.secondaryActionLabels()
	firstWidth := lipgloss.Width(formatButtonLabel(labels[0]))
	secondWidth := lipgloss.Width(formatButtonLabel(labels[1]))
	if got := screen.actionIndexAt(1, firstWidth/2); got != 0 {
		tt.Fatalf("expected import hit to map to index 0, got %d", got)
	}
	exportX := firstWidth + 1 + secondWidth/2
	if got := screen.actionIndexAt(1, exportX); got != 1 {
		tt.Fatalf("expected export hit to map to index 1, got %d", got)
	}
}

func TestModsTabIndexAtUsesRenderedTabWidths(tt *testing.T) {
	loadTestLocalizer(tt)
	screen := modsScreen{tab: modsTabAvailable}
	labels := screen.tabLabels()
	availableWidth := lipgloss.Width(formatModsTabLabel(labels[0], true))
	installedStart := availableWidth + 1
	installedCenter := installedStart + lipgloss.Width(formatModsTabLabel(labels[1], false))/2
	if got := screen.tabIndexAt(installedCenter); got != 1 {
		tt.Fatalf("expected installed-tab hit to map to index 1, got %d", got)
	}
	screen.tab = modsTabInstalled
	installedWidth := lipgloss.Width(formatModsTabLabel(labels[1], true))
	availableCenter := lipgloss.Width(formatModsTabLabel(labels[0], false)) / 2
	if got := screen.tabIndexAt(availableCenter); got != 0 {
		tt.Fatalf("expected available-tab hit to map to index 0 after switching tabs, got %d", got)
	}
	if got := screen.tabIndexAt(lipgloss.Width(formatModsTabLabel(labels[0], false))); got != -1 {
		tt.Fatalf("expected separator gap to stay non-clickable after switching tabs, got %d", got)
	}
	if installedWidth != lipgloss.Width(formatModsTabLabel(labels[1], false)) {
		tt.Fatalf("expected installed tab to reserve equal width across active states")
	}
}

func TestRenderModsTabsKeepsStableLabelPositionsAcrossSelection(tt *testing.T) {
	loadTestLocalizer(tt)
	screen := modsScreen{
		tab:       modsTabAvailable,
		available: make([]manager.ModPackage, 4),
		installed: make([]manager.InstalledMod, 0),
	}
	labels := screen.tabLabels()
	availableActive := ansi.Strip(screen.renderTabs())
	screen.tab = modsTabInstalled
	installedActive := ansi.Strip(screen.renderTabs())
	for _, label := range labels {
		availableParts := strings.SplitN(availableActive, label, 2)
		installedParts := strings.SplitN(installedActive, label, 2)
		if len(availableParts) != 2 || len(installedParts) != 2 {
			tt.Fatalf("expected tab label %q in both renders, got %q and %q", label, availableActive, installedActive)
		}
		if lipgloss.Width(availableParts[0]) != lipgloss.Width(installedParts[0]) {
			tt.Fatalf("expected tab label %q to keep its horizontal anchor, got %q and %q", label, availableActive, installedActive)
		}
	}
}

func TestRenderModsTabUsesSidebarStyleInsteadOfButtonBrackets(tt *testing.T) {
	loadTestLocalizer(tt)
	screen := modsScreen{tab: modsTabAvailable}
	labels := screen.tabLabels()
	tabs := ansi.Strip(screen.renderTabs())
	if !strings.Contains(tabs, "> "+labels[0]+" <") {
		tt.Fatalf("expected active tab to use sidebar-style markers, got %q", tabs)
	}
	if strings.Contains(tabs, formatButtonLabel(labels[0])) || strings.Contains(tabs, formatButtonLabel(labels[1])) {
		tt.Fatalf("expected tab rendering to avoid button brackets, got %q", tabs)
	}
	if !strings.Contains(tabs, labels[1]) {
		tt.Fatalf("expected inactive tab text to remain visible, got %q", tabs)
	}
}

func TestRenderValueControlUsesSharedSelectorShape(tt *testing.T) {
	got := renderValueControl("Steam ID", "123")
	if got != "Steam ID  [ < 123 > ]" {
		tt.Fatalf("expected selector to use shared button framing, got %q", got)
	}
}

func TestRenderSteamUserSelectorUsesThreeAlignedRows(tt *testing.T) {
	loadTestLocalizer(tt)
	got := renderSteamUserSelector(manager.SteamProfile{DisplayName: "Alice", SteamID: "1234567890"}, 3, true, true)
	plain := ansi.Strip(got)
	lines := strings.Split(plain, "\n")
	if len(lines) != 3 {
		tt.Fatalf("expected three-line selector, got %d lines: %q", len(lines), plain)
	}
	if lines[0] != "检测到3个steam用户信息" && lines[0] != "Detected 3 Steam user profiles" {
		tt.Fatalf("unexpected user count line: %q", lines[0])
	}
	if !strings.Contains(lines[1], "切换用户") && !strings.Contains(lines[1], "Switch User") {
		tt.Fatalf("expected selector label on second line, got %q", lines[1])
	}
	if !strings.Contains(lines[1], "[<]") || !strings.Contains(lines[1], "[>]") {
		tt.Fatalf("expected arrow controls on second line, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "steam id") && !strings.Contains(lines[2], "Steam ID") {
		tt.Fatalf("expected steam id label on third line, got %q", lines[2])
	}
	if !strings.Contains(lines[2], "1234567890") {
		tt.Fatalf("expected steam id value on third line, got %q", lines[2])
	}
	if !strings.Contains(got, saveSelectorArrowStyle.Render("[<]")) || !strings.Contains(got, saveSelectorArrowStyle.Render("[>]")) {
		tt.Fatalf("expected arrows to use blue selector style, got %q", got)
	}
}

func TestRenderModListLabelPrefixesRepairBadge(tt *testing.T) {
	label := renderModListLabel(manager.ModPackage{Label: "DamageMeter v1.0.0", NeedsRepair: true}, false)
	if ansi.Strip(label) != "[ ] [?] DamageMeter v1.0.0" {
		tt.Fatalf("expected repair badge in available label, got %q", label)
	}
	if !strings.Contains(label, oldFormatBadgeStyle.Render("[")) || !strings.Contains(label, oldFormatBadgeStyle.Render("?")) || !strings.Contains(label, oldFormatBadgeStyle.Render("]")) {
		tt.Fatalf("expected available repair badge to use old-format badge style, got %q", label)
	}
	installed := renderInstalledModListLabel(manager.InstalledMod{Label: "DamageMeter v1.0.0", NeedsRepair: true}, true)
	if ansi.Strip(installed) != "[x] [?] DamageMeter v1.0.0" {
		tt.Fatalf("expected repair badge in installed label, got %q", installed)
	}
	if !strings.Contains(installed, oldFormatBadgeStyle.Render("[")) || !strings.Contains(installed, oldFormatBadgeStyle.Render("?")) || !strings.Contains(installed, oldFormatBadgeStyle.Render("]")) {
		tt.Fatalf("expected installed repair badge to use old-format badge style, got %q", installed)
	}
}

func TestRenderModListEntryKeepsSelectedColorAfterRepairBadge(tt *testing.T) {
	line := renderModListEntry(manager.ModPackage{Label: "DamageMeter v1.0.0", NeedsRepair: true}, false, true, true)
	if ansi.Strip(line) != "> [ ] [?] DamageMeter v1.0.0" {
		tt.Fatalf("expected selected mod entry text, got %q", ansi.Strip(line))
	}
	if !strings.Contains(line, oldFormatBadgeStyle.Render("[")) || !strings.Contains(line, oldFormatBadgeStyle.Render("?")) || !strings.Contains(line, oldFormatBadgeStyle.Render("]")) {
		tt.Fatalf("expected badge chars to be individually styled red, got %q", line)
	}
	if !strings.Contains(line, focusStyle.Render("DamageMeter v1.0.0")) {
		tt.Fatalf("expected label to retain selected/focused color after badge, got %q", line)
	}
}

func TestRenderModDetailShowsRepairStatus(tt *testing.T) {
	loadTestLocalizer(tt)
	text := renderAvailableModDetail(manager.ModPackage{Label: "DamageMeter", InstallName: "DamageMeter", SourcePath: "C:/mods/DamageMeter", NeedsRepair: true})
	if !strings.Contains(text, t("Repair status: %s", t("Old format suspected"))) {
		tt.Fatalf("expected repair status in detail, got %q", text)
	}
}

func TestModsScreenViewShowsRepairWarningAboveButton(tt *testing.T) {
	loadTestLocalizer(tt)
	screen := modsScreen{
		tab: modsTabAvailable,
		available: []manager.ModPackage{{
			Label:       "DamageMeter",
			InstallName: "DamageMeter",
			SourcePath:  "C:/mods/DamageMeter",
			NeedsRepair: true,
		}},
	}
	app := &appModel{focus: focusContent}
	view := screen.view(app, 120, 20)
	warning := t("This mod format seems incompatible with the new Slay the Spire version. Click to repair.")
	if !strings.Contains(view, errorStyle.Render(warning)) {
		tt.Fatalf("expected repair warning to use error style, got %q", view)
	}
	if strings.Index(ansi.Strip(view), warning) > strings.Index(ansi.Strip(view), formatButtonLabel(t("Repair Mod"))) {
		tt.Fatalf("expected repair warning above button, got %q", ansi.Strip(view))
	}
}

func TestRenderRepairHeaderWrapsWarningText(tt *testing.T) {
	loadTestLocalizer(tt)
	screen := modsScreen{tab: modsTabAvailable, available: []manager.ModPackage{{NeedsRepair: true}}}
	lines := screen.renderRepairHeader(18)
	if len(lines) < 4 {
		tt.Fatalf("expected wrapped warning plus button, got %d lines: %q", len(lines), lines)
	}
	if screen.repairButtonRow(18) != len(lines)-2 {
		tt.Fatalf("expected repair button row to follow wrapped warning lines, got %d for %q", screen.repairButtonRow(18), lines)
	}
	for _, line := range lines[:len(lines)-2] {
		if !strings.Contains(line, errorStyle.Render(ansi.Strip(line))) {
			tt.Fatalf("expected wrapped warning lines to stay red, got %q", line)
		}
	}
}

func TestComputeLayoutStacksHelpBelowMenu(tt *testing.T) {
	m := &appModel{width: 120, height: 40}
	layout := m.computeLayout()
	if layout.menu.frame.width != layout.help.frame.width {
		tt.Fatalf("expected menu/help to share sidebar width, got %d and %d", layout.menu.frame.width, layout.help.frame.width)
	}
	if layout.help.frame.x != 0 || layout.help.frame.y != layout.menu.frame.height {
		tt.Fatalf("expected help below menu in left column, got help frame %+v and menu frame %+v", layout.help.frame, layout.menu.frame)
	}
	if layout.menu.frame.height+layout.help.frame.height != 40 {
		tt.Fatalf("expected sidebar heights to fill full window, got %d", layout.menu.frame.height+layout.help.frame.height)
	}
	if layout.page.frame.x != layout.menu.frame.width || layout.log.frame.x != layout.menu.frame.width {
		tt.Fatalf("expected right panes to start after sidebar, got page %+v log %+v", layout.page.frame, layout.log.frame)
	}
	if layout.page.frame.height+layout.log.frame.height != 40 {
		tt.Fatalf("expected right panes to fill full window height, got %d", layout.page.frame.height+layout.log.frame.height)
	}
}

func TestRenderHelpUsesStructuredSections(tt *testing.T) {
	loadTestLocalizer(tt)
	m := &appModel{current: screenMods}
	global := m.globalHelp()
	if global.Title != "全局：" && global.Title != "Global:" {
		tt.Fatalf("expected structured global title, got %q", global.Title)
	}
	if len(global.Items) != 7 {
		tt.Fatalf("expected 7 structured global help items, got %d", len(global.Items))
	}
	first := renderHelpSection(global)
	if first[0] != renderHelpCells(buildHelpCells(global.Title, helpScopeStyle)) {
		tt.Fatalf("expected scope title to use scope style, got %q", first[0])
	}
	if first[1] != renderHelpActionToken(t("Click"))+renderHelpCells(buildHelpCells(t("menu"), helpTextStyle)) {
		tt.Fatalf("expected first structured action line, got %q", first[1])
	}
	if first[3] != renderHelpActionToken(t("Tab"))+renderHelpCells(buildHelpCells(t("cycle panes"), helpTextStyle)) {
		tt.Fatalf("expected Tab action to use structured rendering, got %q", first[3])
	}
	full := m.renderHelp()
	if !strings.Contains(full, "\n") {
		tt.Fatalf("expected rendered help to span multiple lines, got %q", full)
	}
	if !strings.Contains(full, renderHelpCells(buildHelpCells(t("Mods:"), helpScopeStyle))+"\n"+renderHelpActionToken(t("left/right"))+renderHelpCells(buildHelpCells(t("switch tab"), helpTextStyle))) {
		tt.Fatalf("expected screen-specific help to remain present, got %q", full)
	}
}

func TestRenderHelpActionTokenUsesYellowStyle(tt *testing.T) {
	got := renderHelpActionToken("Ctrl+L")
	if got != helpActionStyle.Render("{Ctrl+L}") {
		tt.Fatalf("expected yellow help token, got %q", got)
	}
}

func TestRenderWrappedHelpItemKeepsContinuationPlain(tt *testing.T) {
	lines := renderWrappedHelpItem(helpItem{Action: "使用", Description: "左侧栏切换页面"}, lipgloss.Width("{使用}左侧栏切换"))
	if len(lines) != 2 {
		tt.Fatalf("expected wrapped help item to span 2 lines, got %d: %q", len(lines), lines)
	}
	if lines[0] != renderHelpActionToken("使用")+renderHelpCells(buildHelpCells("左侧栏切换", helpTextStyle)) {
		tt.Fatalf("expected first line to keep styled token, got %q", lines[0])
	}
	if lines[1] != renderHelpCells(buildHelpCells("页面", helpTextStyle)) {
		tt.Fatalf("expected continuation line to use help text style, got %q", lines[1])
	}
}

func TestCurrentHelpOmitsHomeSection(tt *testing.T) {
	m := &appModel{current: screenHome}
	if got := m.currentHelp(); got.Title != "" || len(got.Items) != 0 {
		tt.Fatalf("expected home screen to omit page-specific help, got %+v", got)
	}
}

func TestNewTranslationKeysResolve(tt *testing.T) {
	loadTestLocalizer(tt)
	checks := map[string]string{
		t("Installed [%d]", 0):                  "已安装[0]",
		t("Type: %s", t("Vanilla")):             "类型：原版",
		t("Slot: %d", 1):                        "槽位：1",
		t("Status: %s", t("(empty)")):           "状态：（空）",
		t("Save Actions"):                       "存档操作",
		t("Steam profiles: %d", 1):              "Steam 档案：1",
		t("Detected %d Steam user profiles", 3): "检测到3个steam用户信息",
		t("Steam ID"):                           "steam id",
		t("%d backups", 0):                      "0个备份",
		t("This mod format seems incompatible with the new Slay the Spire version. Click to repair."): "该模组格式似乎不兼容新版本杀戮尖塔，点击以修复。",
	}
	for got, want := range checks {
		if got != want {
			tt.Fatalf("expected translation %q, got %q", want, got)
		}
	}
}

func TestRenderCopyTargetModalUsesOrangeBorderStyle(tt *testing.T) {
	loadTestLocalizer(tt)
	modal := modalState{
		open:  true,
		title: t("Copy Options"),
		kind:  modalKindCopyTarget,
		copyOptions: []copyTargetOption{
			{Header: true, Label: t("Vanilla Saves")},
			{Label: "2", SaveType: manager.SaveTypeNormal, Slot: 2, Status: t("(empty)")},
		},
		optionCursor: 1,
	}
	view := renderCopyTargetModal(60, 20, modal)
	if !strings.Contains(view, copyModalBorderStyle.Render("│")) {
		tt.Fatalf("expected copy modal body border to use orange style")
	}
	if !strings.Contains(view, copyModalBorderStyle.Render("╰")) {
		tt.Fatalf("expected copy modal footer border to use orange style")
	}
}
