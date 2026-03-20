package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"spiremodgo/internal/manager"
	"spiremodgo/internal/ui/logging"
)

const (
	screenHome     = "home"
	screenMods     = "mods"
	screenSaves    = "saves"
	screenSettings = "settings"
	navQuitIndex   = 4
)

type focusZone int

const (
	focusNav focusZone = iota
	focusContent
	focusLog
)

type App struct {
	program *tea.Program
}

type appModel struct {
	manager  *manager.Manager
	state    *State
	width    int
	height   int
	current  string
	navIndex int
	focus    focusZone
	modal    modalState
	layout   shellLayout
	logs     logging.Model
	home     homeScreen
	mods     modsScreen
	saves    savesScreen
	settings settingsScreen
}

type helpItem struct {
	Action      string
	Description string
}

type helpSection struct {
	Title string
	Items []helpItem
}

type helpCell struct {
	text  string
	style lipgloss.Style
}

func NewApp(mgr *manager.Manager) *App {
	model := newAppModel(mgr)
	return &App{program: tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())}
}

func (a *App) Run() error {
	_, err := a.program.Run()
	return err
}

func newAppModel(mgr *manager.Manager) *appModel {
	_ = loadLocalizer(mgr.BaseDir)
	m := &appModel{
		manager: mgr,
		state:   NewState(mgr),
		current: screenHome,
		focus:   focusNav,
		logs:    logging.New(formatLogEntry),
	}
	m.settings.init()
	m.bootstrap()
	m.refreshCurrentScreen()
	return m
}

func (m *appModel) Init() tea.Cmd {
	return nil
}

func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout = m.computeLayout()
		m.resizeLogViewport()
		return m, nil
	case tea.MouseMsg:
		if m.modal.open {
			return m, m.handleModalMouse(msg)
		}
		return m, m.handleMouse(msg)
	case tea.KeyMsg:
		if m.modal.open {
			return m, m.handleModalKey(msg)
		}
		if m.current == screenSettings && m.settings.editing {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, m.settings.handleKey(m, msg)
		}
		if cmd := m.handleGlobalKey(msg); cmd != nil {
			return m, cmd
		}
		if m.focus == focusNav {
			return m, m.handleNavKey(msg)
		}
		if m.focus == focusLog {
			return m, m.handleLogKey(msg)
		}
		return m, m.handleScreenKey(msg)
	}
	return m, nil
}

func (m *appModel) View() string {
	if m.width == 0 || m.height == 0 {
		return t("Loading...")
	}
	m.layout = m.computeLayout()

	menuBody, menuItems := m.renderSidebar(m.layout.menu.body.width, m.layout.menu.body.height)
	m.layout.menuItems = m.absoluteMenuRects(menuItems)
	menu := renderPanel(t("Menu"), menuBody, m.layout.menu.frame.width, m.layout.menu.frame.height)
	content := m.renderCurrentScreen(m.layout.page.frame.width, m.layout.page.frame.height)
	activity := renderPanel(t("Activity Log"), m.logs.View(), m.layout.log.frame.width, m.layout.log.frame.height)
	help := renderPanel(t("Help"), m.renderHelpBody(m.layout.help.body.width), m.layout.help.frame.width, m.layout.help.frame.height)
	left := strings.Join([]string{menu, help}, "\n")
	right := strings.Join([]string{content, activity}, "\n")
	shell := joinColumns(left, right, m.layout.menu.frame.width, m.layout.page.frame.width)

	if m.modal.open {
		return overlayModal(shell, renderModal(m.width, m.height, m.modal), m.width, m.height)
	}
	return shell
}

func (m *appModel) bootstrap() {
	if dir, err := m.manager.EnsureGameDir(); err != nil {
		m.logError("Game directory detection failed: %v", err)
	} else if dir != "" {
		m.logSuccess("Game directory ready: %s", dir)
	} else {
		m.logWarn("Game directory is not configured yet. Open Settings before install or uninstall actions.")
	}
	if ids, err := m.state.ListSteamIDs(); err != nil {
		m.logError("Steam save scan failed: %v", err)
	} else if len(ids) == 0 {
		m.logWarn("No Steam save directories found under %s", m.manager.SaveRoot)
	} else {
		m.logInfo("Loaded %d Steam save profile(s). Active profile: %s", len(ids), m.state.SelectedSteamID())
	}
}

func (m *appModel) handleGlobalKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c", "q":
		return tea.Quit
	case "ctrl+l":
		locale := toggleLocale()
		m.refreshAllLocalizedState()
		m.logInfo("Language: %s", localeDisplayName(locale))
		return nil
	case "f5":
		m.refreshCurrentScreen()
		m.logInfo("Refreshed %s screen", screenName(m.current))
		return nil
	case "tab":
		switch m.focus {
		case focusNav:
			m.focus = focusContent
		case focusContent:
			m.focus = focusLog
		default:
			m.focus = focusNav
		}
		return nil
	case "shift+tab":
		m.focus = focusNav
		return nil
	case "esc":
		m.focus = focusNav
		if m.current == screenSettings && m.settings.editing {
			m.settings.editing = false
		}
		return nil
	case "1":
		m.switchScreen(screenHome)
	case "2":
		m.switchScreen(screenMods)
	case "3":
		m.switchScreen(screenSaves)
	case "4":
		m.switchScreen(screenSettings)
	}
	return nil
}

func overlayModal(base, overlay string, width, height int) string {
	lineCount := maxInt(height, maxInt(len(strings.Split(base, "\n")), len(strings.Split(overlay, "\n"))))
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")
	merged := make([]string, 0, lineCount)
	for idx := 0; idx < lineCount; idx++ {
		baseLine := ""
		if idx < len(baseLines) {
			baseLine = padVisual(baseLines[idx], width)
		} else {
			baseLine = strings.Repeat(" ", width)
		}
		if idx >= len(overlayLines) {
			merged = append(merged, baseLine)
			continue
		}
		overlayLine := padVisual(overlayLines[idx], width)
		start, end, ok := overlayVisibleSpan(overlayLine, width)
		if !ok {
			merged = append(merged, baseLine)
			continue
		}
		mergedLine := ansi.Cut(baseLine, 0, start) + ansi.Cut(overlayLine, start, end) + ansi.Cut(baseLine, end, width)
		merged = append(merged, mergedLine)
	}
	return strings.Join(merged, "\n")
}

func overlayVisibleSpan(line string, width int) (int, int, bool) {
	start := -1
	end := -1
	for col := 0; col < width; col++ {
		cell := ansi.Strip(ansi.Cut(line, col, col+1))
		if strings.TrimSpace(cell) == "" {
			continue
		}
		if start == -1 {
			start = col
		}
		end = col + 1
	}
	if start == -1 {
		return 0, 0, false
	}
	return start, end, true
}

func (m *appModel) handleNavKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.navIndex > 0 {
			m.navIndex--
		}
	case "down", "j":
		if m.navIndex < navQuitIndex {
			m.navIndex++
		}
	case "enter", "right", "l":
		if m.navIndex == navQuitIndex {
			return tea.Quit
		}
		m.switchScreen(screenByNavIndex(m.navIndex))
		m.focus = focusContent
	}
	return nil
}

func (m *appModel) handleLogKey(msg tea.KeyMsg) tea.Cmd {
	return m.logs.Update(msg)
}

func (m *appModel) handleScreenKey(msg tea.KeyMsg) tea.Cmd {
	switch m.current {
	case screenMods:
		return m.mods.handleKey(m, msg)
	case screenSaves:
		return m.saves.handleKey(m, msg)
	case screenSettings:
		return m.settings.handleKey(m, msg)
	default:
		return nil
	}
}

func (m *appModel) switchScreen(screen string) {
	m.current = screen
	m.navIndex = navIndexByScreen(screen)
	m.refreshCurrentScreen()
}

func (m *appModel) refreshCurrentScreen() {
	switch m.current {
	case screenMods:
		m.mods.refresh(m)
	case screenSaves:
		m.saves.refresh(m)
	case screenSettings:
		m.settings.refresh(m)
	default:
		m.home.refresh(m)
	}
}

func (m *appModel) renderSidebar(width, height int) (string, []rect) {
	return renderMenuBody(m.sidebarItems(), m.navIndex, width, height, m.focus == focusNav)
}

func (m *appModel) sidebarItems() []sidebarItem {
	return []sidebarItem{
		{Label: t("Main Menu")},
		{Label: t("Mod Management")},
		{Label: t("Save Management")},
		{Label: t("Settings")},
		{Label: t("Quit"), Danger: true},
	}
}

func (m *appModel) renderCurrentScreen(width, height int) string {
	switch m.current {
	case screenMods:
		return m.mods.view(m, width, height)
	case screenSaves:
		return m.saves.view(m, width, height)
	case screenSettings:
		return m.settings.view(m, width, height)
	default:
		return m.home.view(m, width, height)
	}
}

func (m *appModel) renderHelp() string {
	sections := []helpSection{m.globalHelp(), m.currentHelp()}
	lines := make([]string, 0, 16)
	for _, section := range sections {
		lines = append(lines, renderHelpSection(section)...)
	}
	return strings.Join(lines, "\n")
}

func (m *appModel) renderHelpBody(width int) string {
	sections := []helpSection{m.globalHelp(), m.currentHelp()}
	lines := make([]string, 0, 16)
	for _, section := range sections {
		lines = append(lines, renderWrappedHelpSection(section, width)...)
	}
	return strings.Join(lines, "\n")
}

func (m *appModel) globalHelp() helpSection {
	return helpSection{
		Title: t("Global:"),
		Items: []helpItem{
			{Action: t("Click"), Description: t("menu")},
			{Action: t("Click"), Description: t("action items")},
			{Action: t("Tab"), Description: t("cycle panes")},
			{Action: t("Esc"), Description: t("return to menu")},
			{Action: t("Ctrl+L"), Description: t("toggle language")},
			{Action: t("F5"), Description: t("refresh")},
			{Action: t("q"), Description: t("quit")},
		},
	}
}

func (m *appModel) currentHelp() helpSection {
	switch m.current {
	case screenMods:
		return m.mods.help()
	case screenSaves:
		return m.saves.help()
	case screenSettings:
		return m.settings.help()
	default:
		return helpSection{}
	}
}

func renderHelpSection(section helpSection) []string {
	lines := make([]string, 0, len(section.Items)+1)
	if section.Title != "" {
		lines = append(lines, renderHelpCells(buildHelpCells(section.Title, helpScopeStyle)))
	}
	for _, item := range section.Items {
		lines = append(lines, renderHelpItem(item))
	}
	return lines
}

func renderWrappedHelpSection(section helpSection, width int) []string {
	lines := make([]string, 0, len(section.Items)+1)
	if section.Title != "" {
		lines = append(lines, wrapHelpCells(buildHelpCells(section.Title, helpScopeStyle), width)...)
	}
	for _, item := range section.Items {
		lines = append(lines, renderWrappedHelpItem(item, width)...)
	}
	return lines
}

func renderHelpItem(item helpItem) string {
	return renderHelpCells(buildHelpItemCells(item))
}

func renderWrappedHelpItem(item helpItem, width int) []string {
	return wrapHelpCells(buildHelpItemCells(item), width)
}

func buildHelpItemCells(item helpItem) []helpCell {
	cells := make([]helpCell, 0, len(item.Action)+len(item.Description)+2)
	if item.Action != "" {
		cells = append(cells, buildHelpCells("{"+item.Action+"}", helpActionStyle)...)
	}
	if item.Description != "" {
		cells = append(cells, buildHelpCells(item.Description, helpTextStyle)...)
	}
	return cells
}

func buildHelpCells(text string, style lipgloss.Style) []helpCell {
	if text == "" {
		return nil
	}
	cells := make([]helpCell, 0, len(text))
	for _, r := range text {
		cells = append(cells, helpCell{text: string(r), style: style})
	}
	return cells
}

func wrapHelpCells(cells []helpCell, width int) []string {
	width = maxInt(1, width)
	if len(cells) == 0 {
		return nil
	}
	lines := make([]string, 0, 4)
	line := make([]helpCell, 0, width)
	used := 0
	flush := func() {
		if len(line) == 0 {
			return
		}
		lines = append(lines, renderHelpCells(line))
		line = make([]helpCell, 0, width)
		used = 0
	}
	for _, cell := range cells {
		cellWidth := lipgloss.Width(cell.text)
		if cellWidth == 0 {
			line = append(line, cell)
			continue
		}
		if used > 0 && used+cellWidth > width {
			flush()
		}
		line = append(line, cell)
		used += cellWidth
	}
	flush()
	return lines
}

func renderHelpCells(cells []helpCell) string {
	var builder strings.Builder
	for _, cell := range cells {
		builder.WriteString(cell.style.Render(cell.text))
	}
	return builder.String()
}

func renderHelpActionToken(action string) string {
	return renderHelpCells(buildHelpCells("{"+action+"}", helpActionStyle))
}

func (m *appModel) resizeLogViewport() {
	if m.width == 0 || m.height == 0 {
		return
	}
	m.layout = m.computeLayout()
	m.logs.Resize(m.layout.log.body.width, m.layout.log.body.height)
}

func (m *appModel) computeLayout() shellLayout {
	width := maxInt(40, m.width)
	height := maxInt(12, m.height)
	menuWidth := clampInt(width/5, 18, 26)
	if width < 80 {
		menuWidth = clampInt(width/4, 16, 22)
	}
	rightWidth := maxInt(20, width-menuWidth)
	menuHeight := maxInt(7, (height+1)/2)
	helpHeight := maxInt(3, height-menuHeight)
	menuHeight = height - helpHeight
	logHeight := clampInt(height/4, 4, 8)
	pageHeight := height - logHeight
	if pageHeight < 6 {
		pageHeight = 6
		logHeight = maxInt(3, height-pageHeight)
	}
	return shellLayout{
		menu: newPanelLayout(0, 0, menuWidth, menuHeight),
		page: newPanelLayout(menuWidth, 0, rightWidth, pageHeight),
		log:  newPanelLayout(menuWidth, pageHeight, rightWidth, logHeight),
		help: newPanelLayout(0, menuHeight, menuWidth, helpHeight),
	}
}

func (m *appModel) showInfo(title, body string) {
	m.modal = modalState{open: true, title: t(title), body: t(body), kind: modalKindInfo}
}

func (m *appModel) showConfirm(title, body string, onConfirm func(*appModel)) {
	m.modal = modalState{open: true, title: t(title), body: t(body), kind: modalKindConfirm, onConfirm: onConfirm}
}

func (m *appModel) showCopyTargetModal(title string, options []copyTargetOption, onPick func(*appModel, copyTargetOption, bool)) {
	cursor := 0
	for idx, option := range options {
		if !option.Header {
			cursor = idx
			break
		}
	}
	m.modal = modalState{open: true, title: t(title), kind: modalKindCopyTarget, copyOptions: options, optionCursor: cursor, onPickCopy: onPick}
}

func (m *appModel) handleModalKey(msg tea.KeyMsg) tea.Cmd {
	if m.modal.kind == modalKindCopyTarget {
		switch msg.String() {
		case "up", "k":
			m.modal.optionCursor = moveModalOption(m.modal.copyOptions, m.modal.optionCursor, -1)
		case "down", "j":
			m.modal.optionCursor = moveModalOption(m.modal.copyOptions, m.modal.optionCursor, 1)
		case "left", "h":
			if m.modal.actionCursor > 0 {
				m.modal.actionCursor--
			}
		case "right", "l", "tab":
			if m.modal.actionCursor < 2 {
				m.modal.actionCursor++
			} else {
				m.modal.actionCursor = 0
			}
		case "enter":
			return m.executeCopyModalAction()
		case "esc", "q":
			m.modal = modalState{}
		}
		return nil
	}
	switch msg.String() {
	case "enter", "y":
		confirm := m.modal.kind == modalKindConfirm
		handler := m.modal.onConfirm
		m.modal = modalState{}
		if confirm && handler != nil {
			handler(m)
		}
	case "esc", "n", "q":
		m.modal = modalState{}
	}
	return nil
}

func (m *appModel) handleModalMouse(msg tea.MouseMsg) tea.Cmd {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil
	}
	layout := modalLayout(m.width, m.height, m.modal)
	if m.modal.kind == modalKindCopyTarget {
		for idx, optionRect := range layout.optionRects {
			if optionRect.contains(msg.X, msg.Y) {
				m.modal.optionCursor = moveModalOption(m.modal.copyOptions, layout.optionIndexes[idx], 0)
				return nil
			}
		}
		if layout.primaryButton.contains(msg.X, msg.Y) {
			m.modal.actionCursor = 0
			return m.executeCopyModalAction()
		}
		if layout.secondaryButton.contains(msg.X, msg.Y) {
			m.modal.actionCursor = 1
			return m.executeCopyModalAction()
		}
		if layout.cancelButton.contains(msg.X, msg.Y) {
			m.modal = modalState{}
		}
		return nil
	}
	if layout.primaryButton.contains(msg.X, msg.Y) {
		handler := m.modal.onConfirm
		confirm := m.modal.kind == modalKindConfirm
		m.modal = modalState{}
		if confirm && handler != nil {
			handler(m)
		}
		return nil
	}
	if layout.cancelButton.contains(msg.X, msg.Y) {
		m.modal = modalState{}
	}
	return nil
}

func (m *appModel) executeCopyModalAction() tea.Cmd {
	if len(m.modal.copyOptions) == 0 || m.modal.optionCursor >= len(m.modal.copyOptions) {
		m.modal = modalState{}
		return nil
	}
	if m.modal.actionCursor == 2 {
		m.modal = modalState{}
		return nil
	}
	option := m.modal.copyOptions[m.modal.optionCursor]
	if option.Header {
		m.modal.optionCursor = moveModalOption(m.modal.copyOptions, m.modal.optionCursor, 1)
		return nil
	}
	pick := m.modal.onPickCopy
	backupCopy := m.modal.actionCursor == 1
	m.modal = modalState{}
	if pick != nil {
		pick(m, option, backupCopy)
	}
	return nil
}

func moveModalOption(options []copyTargetOption, start, delta int) int {
	if len(options) == 0 {
		return 0
	}
	if delta == 0 {
		if start >= 0 && start < len(options) && !options[start].Header {
			return start
		}
		delta = 1
	}
	idx := start
	for range len(options) {
		idx = (idx + delta + len(options)) % len(options)
		if !options[idx].Header {
			return idx
		}
	}
	return start
}

func (m *appModel) handleMouse(msg tea.MouseMsg) tea.Cmd {
	if msg.Action != tea.MouseActionPress && !tea.MouseEvent(msg).IsWheel() {
		return nil
	}
	if m.layout.menu.body.contains(msg.X, msg.Y) {
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			m.focus = focusNav
			for idx, item := range m.layout.menuItems {
				if item.contains(msg.X, msg.Y) {
					if idx == navQuitIndex {
						if m.navIndex == navQuitIndex {
							return tea.Quit
						}
						m.navIndex = navQuitIndex
						break
					}
					m.navIndex = idx
					m.switchScreen(screenByNavIndex(idx))
					m.focus = focusContent
					break
				}
			}
		}
		return nil
	}
	if m.layout.log.body.contains(msg.X, msg.Y) {
		m.focus = focusLog
		return m.logs.Update(msg)
	}
	if m.layout.page.frame.contains(msg.X, msg.Y) {
		if msg.Action == tea.MouseActionPress {
			m.focus = focusContent
		}
		localX, localY := m.layout.page.frame.local(msg.X, msg.Y)
		return m.handleScreenMouse(msg, localX, localY, m.layout.page.frame.width, m.layout.page.frame.height)
	}
	return nil
}

func (m *appModel) handleScreenMouse(msg tea.MouseMsg, x, y, width, height int) tea.Cmd {
	switch m.current {
	case screenMods:
		return m.mods.handleMouse(m, msg, x, y, width, height)
	case screenSaves:
		return m.saves.handleMouse(m, msg, x, y, width, height)
	case screenSettings:
		return m.settings.handleMouse(m, msg, x, y, width, height)
	default:
		return nil
	}
}

func (m *appModel) showError(action string, err error) {
	if err == nil {
		return
	}
	localizedAction := t(action)
	localizedErr := m.localizeError(err)
	m.logError("%s: %s", localizedAction, localizedErr)
	m.showInfo(t("Error"), localizedAction+"\n\n"+localizedErr)
}

func (m *appModel) logInfo(format string, args ...any) {
	m.logs.Add("info", format, args...)
}

func (m *appModel) logSuccess(format string, args ...any) {
	m.logs.Add("ok", format, args...)
}

func (m *appModel) logWarn(format string, args ...any) {
	m.logs.Add("warn", format, args...)
}

func (m *appModel) logError(format string, args ...any) {
	m.logs.Add("error", format, args...)
}

func formatLogEntry(stamp, level, key string, args ...any) string {
	label := level
	switch level {
	case "ok":
		label = okStyle.Render(t("OK"))
	case "warn":
		label = warnStyle.Render(t("WARN"))
	case "error":
		label = errorStyle.Render(t("ERR"))
	default:
		label = mutedStyle.Render(t("INFO"))
	}
	return fmt.Sprintf("%s  [%s] %s", mutedStyle.Render(stamp), label, t(key, args...))
}

func screenName(screen string) string {
	switch screen {
	case screenMods:
		return t("Mod Management")
	case screenSaves:
		return t("Save Management")
	case screenSettings:
		return t("Settings")
	default:
		return t("Main Menu")
	}
}

func (m *appModel) refreshAllLocalizedState() {
	m.home.refresh(m)
	m.mods.refresh(m)
	m.saves.refresh(m)
	m.settings.refresh(m)
	m.logs.Sync()
}

func (m *appModel) localizeError(err error) string {
	if err == nil {
		return ""
	}
	text := err.Error()
	const steamPrefix = "no Steam save directories found in "
	if strings.HasPrefix(text, steamPrefix) {
		return t("no Steam save directories found in %s", strings.TrimPrefix(text, steamPrefix))
	}
	return t(text)
}

func localeDisplayName(locale string) string {
	if locale == localeZhCN {
		return t("Chinese")
	}
	return t("English")
}

func screenByNavIndex(index int) string {
	switch index {
	case 1:
		return screenMods
	case 2:
		return screenSaves
	case 3:
		return screenSettings
	default:
		return screenHome
	}
}

func navIndexByScreen(screen string) int {
	switch screen {
	case screenMods:
		return 1
	case screenSaves:
		return 2
	case screenSettings:
		return 3
	default:
		return 0
	}
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func joinColumns(left, right string, leftWidth, rightWidth int) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	height := maxInt(len(leftLines), len(rightLines))
	lines := make([]string, 0, height)
	for i := 0; i < height; i++ {
		leftLine := strings.Repeat(" ", leftWidth)
		rightLine := strings.Repeat(" ", rightWidth)
		if i < len(leftLines) {
			leftLine = padVisual(leftLines[i], leftWidth)
		}
		if i < len(rightLines) {
			rightLine = padVisual(rightLines[i], rightWidth)
		}
		lines = append(lines, leftLine+rightLine)
	}
	return strings.Join(lines, "\n")
}

func (m *appModel) absoluteMenuRects(local []rect) []rect {
	items := make([]rect, 0, len(local))
	for _, item := range local {
		items = append(items, rect{x: m.layout.menu.body.x + item.x, y: m.layout.menu.body.y + item.y, width: item.width, height: item.height})
	}
	return items
}
