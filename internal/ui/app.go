package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"slaymodgo/internal/manager"
)

const (
	screenHome      = "home"
	screenInstall   = "install"
	screenUninstall = "uninstall"
	screenInstalled = "installed"
	screenSaves     = "saves"
	screenSettings  = "settings"
)

type focusZone int

const (
	focusNav focusZone = iota
	focusContent
)

type App struct {
	program *tea.Program
}

type appModel struct {
	manager   *manager.Manager
	state     *State
	width     int
	height    int
	current   string
	navIndex  int
	focus     focusZone
	modal     modalState
	logs      logModel
	home      homeScreen
	install   installScreen
	uninstall uninstallScreen
	installed installedScreen
	saves     savesScreen
	settings  settingsScreen
}

func NewApp(mgr *manager.Manager) *App {
	model := newAppModel(mgr)
	return &App{program: tea.NewProgram(model, tea.WithAltScreen())}
}

func (a *App) Run() error {
	_, err := a.program.Run()
	return err
}

func newAppModel(mgr *manager.Manager) *appModel {
	m := &appModel{
		manager: mgr,
		state:   NewState(mgr),
		current: screenHome,
		focus:   focusNav,
		logs:    newLogModel(),
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
		m.resizeLogViewport()
		return m, nil
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
			m.handleNavKey(msg)
			return m, nil
		}
		return m, m.handleScreenKey(msg)
	}
	return m, nil
}

func (m *appModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	sidebarWidth, mainWidth, contentHeight, logHeight, helpHeight := m.layoutMetrics()
	sidebar := sidebarStyle.Width(sidebarWidth).Height(m.height).Render(m.renderSidebar(maxInt(18, sidebarWidth-3), maxInt(6, m.height-2)))

	contentBodyHeight := maxInt(4, contentHeight-2)
	logBodyHeight := maxInt(1, logHeight-2)
	m.logs.Resize(maxInt(20, mainWidth-2), logBodyHeight)

	content := renderFlatSection(screenName(m.current), m.renderCurrentScreen(maxInt(24, mainWidth-2), contentBodyHeight), mainWidth, contentHeight)
	activity := renderFlatSection("Activity Log", m.logs.View(), mainWidth, logHeight)
	help := renderFlatSection("Help", m.renderHelp(), mainWidth, helpHeight)
	workspace := lipgloss.JoinVertical(lipgloss.Left, content, activity, help)
	shell := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, workspace)

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
	case "f5":
		m.refreshCurrentScreen()
		m.logInfo("Refreshed %s screen", screenName(m.current))
		return nil
	case "tab":
		if m.focus == focusNav {
			m.focus = focusContent
		} else {
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
		m.switchScreen(screenInstall)
	case "3":
		m.switchScreen(screenUninstall)
	case "4":
		m.switchScreen(screenInstalled)
	case "5":
		m.switchScreen(screenSaves)
	case "6":
		m.switchScreen(screenSettings)
	}
	return nil
}

func overlayModal(base, overlay string, width, height int) string {
	basePlaced := lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, base)
	overlayPlaced := lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, overlay)
	baseLines := strings.Split(basePlaced, "\n")
	overlayLines := strings.Split(overlayPlaced, "\n")
	if len(baseLines) < len(overlayLines) {
		missing := len(overlayLines) - len(baseLines)
		for i := 0; i < missing; i++ {
			baseLines = append(baseLines, "")
		}
	}
	for idx, line := range overlayLines {
		if strings.TrimSpace(line) != "" {
			baseLines[idx] = line
		}
	}
	return strings.Join(baseLines, "\n")
}

func (m *appModel) handleNavKey(msg tea.KeyMsg) {
	switch msg.String() {
	case "up", "k":
		if m.navIndex > 0 {
			m.navIndex--
		}
	case "down", "j":
		if m.navIndex < 5 {
			m.navIndex++
		}
	case "enter", "right", "l":
		m.switchScreen(screenByNavIndex(m.navIndex))
		m.focus = focusContent
	}
}

func (m *appModel) handleScreenKey(msg tea.KeyMsg) tea.Cmd {
	switch m.current {
	case screenInstall:
		return m.install.handleKey(m, msg)
	case screenUninstall:
		return m.uninstall.handleKey(m, msg)
	case screenInstalled:
		return m.installed.handleKey(m, msg)
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
	case screenInstall:
		m.install.refresh(m)
	case screenUninstall:
		m.uninstall.refresh(m)
	case screenInstalled:
		m.installed.refresh(m)
	case screenSaves:
		m.saves.refresh(m)
	case screenSettings:
		m.settings.refresh(m)
	default:
		m.home.refresh(m)
	}
}

func (m *appModel) renderSidebar(width, height int) string {
	gameDir := m.state.GameDir()
	if gameDir == "" {
		gameDir = "(not configured)"
	}
	steamID := m.state.SelectedSteamID()
	if steamID == "" {
		steamID = "(none)"
	}
	items := []string{
		"1. Main Menu",
		"2. Install Mods",
		"3. Uninstall Mods",
		"4. Installed Mods",
		"5. Save Management",
		"6. Settings",
	}
	lines := make([]string, 0, len(items)+14)
	lines = append(lines,
		titleStyle.Render("Slay the Spire 2"),
		accentStyle.Render("Mod Manager"),
		"",
		sectionTitleStyle.Render("Current Page"),
		navActiveStyle.Render(screenName(m.current)),
		"",
		sectionTitleStyle.Render("Context"),
		metaLabelStyle.Render("Game Dir"),
		okStyle.Render(gameDir),
		"",
		metaLabelStyle.Render("Steam ID"),
		okStyle.Render(steamID),
		"",
		sectionTitleStyle.Render("Pages"),
	)
	for idx, item := range items {
		prefix := "  "
		style := navItemStyle
		if idx == m.navIndex {
			prefix = "> "
			style = navActiveStyle
		}
		if m.focus == focusNav && idx == m.navIndex {
			style = navFocusStyle
		}
		lines = append(lines, style.Render(prefix+item))
	}
	lines = append(lines,
		"",
		sectionTitleStyle.Render("Keys"),
		mutedStyle.Render("Up/Down select page"),
		mutedStyle.Render("Enter open page"),
		mutedStyle.Render("Tab switch focus"),
		mutedStyle.Render("F5 refresh | q quit"),
	)
	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().Width(width).Height(height).Render(content)
}

func (m *appModel) renderCurrentScreen(width, height int) string {
	switch m.current {
	case screenInstall:
		return m.install.view(m, width, height)
	case screenUninstall:
		return m.uninstall.view(m, width, height)
	case screenInstalled:
		return m.installed.view(m, width, height)
	case screenSaves:
		return m.saves.view(m, width, height)
	case screenSettings:
		return m.settings.view(m, width, height)
	default:
		return m.home.view(m, width, height)
	}
}

func (m *appModel) renderHelp() string {
	common := "Global: sidebar navigation | Tab focus content | Shift+Tab/Esc return to sidebar | F5 refresh | q quit"
	screenHelp := ""
	switch m.current {
	case screenInstall:
		screenHelp = m.install.help()
	case screenUninstall:
		screenHelp = m.uninstall.help()
	case screenInstalled:
		screenHelp = m.installed.help()
	case screenSaves:
		screenHelp = m.saves.help()
	case screenSettings:
		screenHelp = m.settings.help()
	default:
		screenHelp = m.home.help()
	}
	return common + "\n" + screenHelp
}

func (m *appModel) resizeLogViewport() {
	if m.width == 0 || m.height == 0 {
		return
	}
	_, mainWidth, _, logHeight, _ := m.layoutMetrics()
	m.logs.Resize(maxInt(20, mainWidth-2), maxInt(1, logHeight-2))
}

func (m *appModel) layoutMetrics() (int, int, int, int, int) {
	sidebarWidth := clampInt(m.width/4, 24, 30)
	if m.width < 84 {
		sidebarWidth = clampInt(m.width/3, 20, 24)
	}
	mainWidth := maxInt(30, m.width-sidebarWidth)
	helpHeight := 4
	logHeight := clampInt(m.height/4, 5, 8)
	contentHeight := m.height - logHeight - helpHeight
	if contentHeight < 8 {
		shortfall := 8 - contentHeight
		if logHeight > 4 {
			shrink := minInt(shortfall, logHeight-4)
			logHeight -= shrink
			shortfall -= shrink
		}
		if shortfall > 0 && helpHeight > 3 {
			shrink := minInt(shortfall, helpHeight-3)
			helpHeight -= shrink
		}
		contentHeight = maxInt(6, m.height-logHeight-helpHeight)
	}
	return sidebarWidth, mainWidth, contentHeight, logHeight, helpHeight
}

func (m *appModel) showInfo(title, body string) {
	m.modal = modalState{open: true, title: title, body: body}
}

func (m *appModel) showConfirm(title, body string, onConfirm func(*appModel)) {
	m.modal = modalState{open: true, title: title, body: body, confirm: true, onConfirm: onConfirm}
}

func (m *appModel) handleModalKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter", "y":
		confirm := m.modal.confirm
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

func (m *appModel) showError(action string, err error) {
	if err == nil {
		return
	}
	m.logError("%s: %v", action, err)
	m.showInfo("Error", fmt.Sprintf("%s\n\n%v", action, err))
}

func (m *appModel) logInfo(format string, args ...any) {
	m.logs.Add("info", fmt.Sprintf(format, args...))
}

func (m *appModel) logSuccess(format string, args ...any) {
	m.logs.Add("ok", fmt.Sprintf(format, args...))
}

func (m *appModel) logWarn(format string, args ...any) {
	m.logs.Add("warn", fmt.Sprintf(format, args...))
}

func (m *appModel) logError(format string, args ...any) {
	m.logs.Add("error", fmt.Sprintf(format, args...))
}

func screenName(screen string) string {
	switch screen {
	case screenInstall:
		return "Install Mods"
	case screenUninstall:
		return "Uninstall Mods"
	case screenInstalled:
		return "Installed Mods"
	case screenSaves:
		return "Save Management"
	case screenSettings:
		return "Settings"
	default:
		return "Main Menu"
	}
}

func screenByNavIndex(index int) string {
	switch index {
	case 1:
		return screenInstall
	case 2:
		return screenUninstall
	case 3:
		return screenInstalled
	case 4:
		return screenSaves
	case 5:
		return screenSettings
	default:
		return screenHome
	}
}

func navIndexByScreen(screen string) int {
	switch screen {
	case screenInstall:
		return 1
	case screenUninstall:
		return 2
	case screenInstalled:
		return 3
	case screenSaves:
		return 4
	case screenSettings:
		return 5
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
