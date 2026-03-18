package ui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle          = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")).Padding(0, 1)
	headerStyle         = lipgloss.NewStyle().Padding(0, 1)
	footerStyle         = lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("250"))
	sidebarStyle        = lipgloss.NewStyle().Padding(1, 1).BorderStyle(lipgloss.NormalBorder()).BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(true).BorderForeground(lipgloss.Color("238"))
	workspaceStyle      = lipgloss.NewStyle().Padding(0, 1)
	sectionTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	sectionDividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	sectionBodyStyle    = lipgloss.NewStyle().Padding(0, 1)
	metaLabelStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
	accentStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
	mutedStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	okStyle             = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	warnStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("221"))
	errorStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	navItemStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	navActiveStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
	navFocusStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("31")).Padding(0, 1)
	modalBoxStyle       = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(lipgloss.Color("81")).Padding(1, 2).Background(lipgloss.Color("236")).Foreground(lipgloss.Color("230"))
	cursorStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
	selectedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	focusStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("221"))
)
