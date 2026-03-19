package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type modalState struct {
	open      bool
	title     string
	body      string
	confirm   bool
	onConfirm func(*appModel)
}

type modalGeometry struct {
	frame         rect
	confirmButton rect
	closeButton   rect
}

func modalLayout(width, height int, modal modalState) modalGeometry {
	boxWidth := clampInt(width/2, 40, maxInt(40, width-6))
	boxHeight := clampInt(height/3, 8, maxInt(8, height-4))
	x := maxInt(0, (width-boxWidth)/2)
	y := maxInt(0, (height-boxHeight)/2)
	frame := rect{x: x, y: y, width: boxWidth, height: boxHeight}
	body := newPanelLayout(x, y, boxWidth, boxHeight).body
	buttonY := body.y + maxInt(0, body.height-1)
	confirmRect := rect{x: body.x, y: buttonY, width: 0, height: 0}
	closeRect := rect{x: body.x, y: buttonY, width: 0, height: 0}
	confirmLabel := "[ " + t("Confirm") + " ]"
	if !modal.confirm {
		confirmLabel = "[ " + t("Close") + " ]"
	}
	closeLabel := "[ " + t("Cancel") + " ]"
	confirmRect.width = lipgloss.Width(confirmLabel)
	confirmRect.height = 1
	closeRect.width = lipgloss.Width(closeLabel)
	closeRect.height = 1
	confirmRect.x = body.x
	if modal.confirm {
		closeRect.x = body.x + confirmRect.width + 2
	} else {
		closeRect = confirmRect
	}
	return modalGeometry{frame: frame, confirmButton: confirmRect, closeButton: closeRect}
}

func renderModal(width, height int, modal modalState) string {
	if !modal.open {
		return ""
	}
	layout := modalLayout(width, height, modal)
	buttonLine := buttonActiveStyle.Render("[ "+t("Confirm")+" ]") + "  " + buttonStyle.Render("[ "+t("Cancel")+" ]")
	if !modal.confirm {
		buttonLine = buttonActiveStyle.Render("[ " + t("Close") + " ]")
	}
	bodyHeight := newPanelLayout(layout.frame.x, layout.frame.y, layout.frame.width, layout.frame.height).body.height
	textLines := strings.Split(modal.body, "\n")
	maxTextLines := maxInt(0, bodyHeight-1)
	if len(textLines) > maxTextLines {
		textLines = textLines[:maxTextLines]
	}
	for len(textLines) < maxTextLines {
		textLines = append(textLines, "")
	}
	bodyLines := append(textLines, buttonLine)
	body := strings.Join(bodyLines, "\n")
	box := renderPanel(modal.title, body, layout.frame.width, layout.frame.height)
	canvas := make([]string, 0, height)
	for i := 0; i < layout.frame.y && len(canvas) < height; i++ {
		canvas = append(canvas, strings.Repeat(" ", width))
	}
	for _, line := range strings.Split(box, "\n") {
		if len(canvas) >= height {
			break
		}
		canvas = append(canvas, strings.Repeat(" ", layout.frame.x)+padVisual(line, layout.frame.width)+strings.Repeat(" ", maxInt(0, width-layout.frame.x-layout.frame.width)))
	}
	for len(canvas) < height {
		canvas = append(canvas, strings.Repeat(" ", width))
	}
	return strings.Join(canvas, "\n")
}
