package ui

import "github.com/charmbracelet/lipgloss"

type modalState struct {
	open      bool
	title     string
	body      string
	confirm   bool
	onConfirm func(*appModel)
}

func renderModal(width, height int, modal modalState) string {
	if !modal.open {
		return ""
	}
	footer := "Enter confirm"
	if !modal.confirm {
		footer = "Enter close"
	}
	footer += " | Esc cancel"
	content := sectionTitleStyle.Render(modal.title) + "\n\n" + modal.body + "\n\n" + mutedStyle.Render(footer)
	box := modalBoxStyle.Width(clampInt(width/2, 40, maxInt(40, width-6))).Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
