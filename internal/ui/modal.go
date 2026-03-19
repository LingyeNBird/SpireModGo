package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"slaymodgo/internal/manager"
)

type modalKind int

const (
	modalKindInfo modalKind = iota
	modalKindConfirm
	modalKindCopyTarget
)

type copyTargetOption struct {
	Header   bool
	Label    string
	SaveType manager.SaveType
	Slot     int
	Status   string
	HasData  bool
}

type modalState struct {
	open         bool
	title        string
	body         string
	kind         modalKind
	onConfirm    func(*appModel)
	copyOptions  []copyTargetOption
	optionCursor int
	actionCursor int
	onPickCopy   func(*appModel, copyTargetOption, bool)
}

type modalGeometry struct {
	frame           rect
	optionRects     []rect
	optionIndexes   []int
	primaryButton   rect
	secondaryButton rect
	cancelButton    rect
}

func modalLayout(width, height int, modal modalState) modalGeometry {
	boxWidth := clampInt(width/2, 44, maxInt(44, width-6))
	minHeight := 10
	if modal.kind == modalKindCopyTarget {
		minHeight = 12
	}
	boxHeight := clampInt(height/2, minHeight, maxInt(minHeight, height-4))
	x := maxInt(0, (width-boxWidth)/2)
	y := maxInt(0, (height-boxHeight)/2)
	frame := rect{x: x, y: y, width: boxWidth, height: boxHeight}
	body := newPanelLayout(x, y, boxWidth, boxHeight).body
	buttonY := body.y + maxInt(0, body.height-1)
	if modal.kind == modalKindCopyTarget {
		buttonY = frame.y + frame.height - 1
	}
	geom := modalGeometry{frame: frame}
	if modal.kind == modalKindCopyTarget {
		optionRows := copyOptionRowIndexes(modal, body.height-1)
		for _, row := range optionRows {
			if row.selectable {
				geom.optionRects = append(geom.optionRects, rect{x: body.x, y: body.y + row.lineIndex, width: body.width, height: 1})
				geom.optionIndexes = append(geom.optionIndexes, row.optionIndex)
			}
		}
	}
	primaryLabel := lipgloss.Width("[ " + primaryButtonLabel(modal) + " ]")
	secondaryLabel := lipgloss.Width("[ " + secondaryButtonLabel(modal) + " ]")
	cancelLabel := lipgloss.Width("[ " + t("Cancel") + " ]")
	geom.primaryButton = rect{x: body.x, y: buttonY, width: primaryLabel, height: 1}
	if modal.kind == modalKindCopyTarget {
		geom.secondaryButton = rect{x: body.x + primaryLabel + 2, y: buttonY, width: secondaryLabel, height: 1}
		geom.cancelButton = rect{x: geom.secondaryButton.x + secondaryLabel + 2, y: buttonY, width: cancelLabel, height: 1}
	} else if modal.kind == modalKindConfirm {
		geom.cancelButton = rect{x: body.x + primaryLabel + 2, y: buttonY, width: cancelLabel, height: 1}
	} else {
		geom.cancelButton = geom.primaryButton
	}
	return geom
}

type copyOptionRow struct {
	lineIndex   int
	optionIndex int
	selectable  bool
}

func copyOptionRowIndexes(modal modalState, maxTextLines int) []copyOptionRow {
	rows := []copyOptionRow{{lineIndex: 0, optionIndex: -1, selectable: false}}
	lineIndex := 1
	for idx, option := range modal.copyOptions {
		if lineIndex >= maxTextLines {
			break
		}
		rows = append(rows, copyOptionRow{lineIndex: lineIndex, optionIndex: idx, selectable: !option.Header})
		lineIndex++
	}
	return rows
}

func primaryButtonLabel(modal modalState) string {
	switch modal.kind {
	case modalKindCopyTarget:
		return t("Copy")
	case modalKindConfirm:
		return t("Confirm")
	default:
		return t("Close")
	}
}

func secondaryButtonLabel(modal modalState) string {
	if modal.kind == modalKindCopyTarget {
		return t("Backup and Copy")
	}
	return t("Cancel")
}

func renderModal(width, height int, modal modalState) string {
	if !modal.open {
		return ""
	}
	if modal.kind == modalKindCopyTarget {
		return renderCopyTargetModal(width, height, modal)
	}
	layout := modalLayout(width, height, modal)
	bodyHeight := newPanelLayout(layout.frame.x, layout.frame.y, layout.frame.width, layout.frame.height).body.height
	textLines := modalTextLines(modal, bodyHeight-1)
	buttonLine := modalButtonLine(modal)
	body := strings.Join(append(textLines, buttonLine), "\n")
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

func renderCopyTargetModal(width, height int, modal modalState) string {
	layout := modalLayout(width, height, modal)
	bodyHeight := maxInt(1, layout.frame.height-2)
	textLines := modalTextLines(modal, bodyHeight)
	bodyLines := normalizeLines(strings.Join(textLines, "\n"), layout.frame.width-2, bodyHeight)
	lines := make([]string, 0, height)
	for i := 0; i < layout.frame.y && len(lines) < height; i++ {
		lines = append(lines, strings.Repeat(" ", width))
	}
	lines = append(lines, strings.Repeat(" ", layout.frame.x)+makeTopBorder(modal.title, layout.frame.width)+strings.Repeat(" ", maxInt(0, width-layout.frame.x-layout.frame.width)))
	for _, line := range bodyLines {
		if len(lines) >= height {
			break
		}
		content := borderStyle.Render("│") + panelBodyStyle.Render(line) + borderStyle.Render("│")
		lines = append(lines, strings.Repeat(" ", layout.frame.x)+content+strings.Repeat(" ", maxInt(0, width-layout.frame.x-layout.frame.width)))
	}
	bottom := renderBorderButtonRow([]string{t("Copy"), t("Backup and Copy"), t("Cancel")}, layout.frame.width, modal.actionCursor)
	if len(lines) < height {
		lines = append(lines, strings.Repeat(" ", layout.frame.x)+bottom+strings.Repeat(" ", maxInt(0, width-layout.frame.x-layout.frame.width)))
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}

func modalTextLines(modal modalState, maxTextLines int) []string {
	if maxTextLines < 1 {
		return []string{}
	}
	if modal.kind != modalKindCopyTarget {
		lines := strings.Split(modal.body, "\n")
		if len(lines) > maxTextLines {
			lines = lines[:maxTextLines]
		}
		for len(lines) < maxTextLines {
			lines = append(lines, "")
		}
		return lines
	}
	lines := []string{t("Choose a destination save slot")}
	for idx, option := range modal.copyOptions {
		if len(lines) >= maxTextLines {
			break
		}
		if option.Header {
			lines = append(lines, option.Label)
			continue
		}
		prefix := "  "
		if idx == modal.optionCursor {
			prefix = "> "
		}
		lines = append(lines, prefix+option.Label+"  "+option.Status)
	}
	for len(lines) < maxTextLines {
		lines = append(lines, "")
	}
	return lines
}

func modalButtonLine(modal modalState) string {
	primary := renderActionLine(primaryButtonLabel(modal), modal.actionCursor == 0)
	if modal.kind == modalKindCopyTarget {
		secondary := renderActionLine(secondaryButtonLabel(modal), modal.actionCursor == 1)
		cancel := renderActionLine(t("Cancel"), modal.actionCursor == 2)
		return strings.Join([]string{primary, secondary, cancel}, " ")
	}
	if modal.kind == modalKindConfirm {
		cancel := renderActionLine(t("Cancel"), modal.actionCursor == 1)
		return strings.Join([]string{primary, cancel}, " ")
	}
	return primary
}
