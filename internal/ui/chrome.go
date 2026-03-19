package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type rect struct {
	x      int
	y      int
	width  int
	height int
}

func (r rect) contains(x, y int) bool {
	return x >= r.x && x < r.x+r.width && y >= r.y && y < r.y+r.height
}

func (r rect) local(x, y int) (int, int) {
	return x - r.x, y - r.y
}

type panelLayout struct {
	frame rect
	body  rect
}

func newPanelLayout(x, y, width, height int) panelLayout {
	if width < 2 {
		width = 2
	}
	if height < 2 {
		height = 2
	}
	return panelLayout{
		frame: rect{x: x, y: y, width: width, height: height},
		body:  rect{x: x + 1, y: y + 1, width: maxInt(1, width-2), height: maxInt(1, height-2)},
	}
}

type shellLayout struct {
	menu      panelLayout
	page      panelLayout
	log       panelLayout
	help      panelLayout
	menuItems []rect
}

type splitBodyLayout struct {
	leftHeader  rect
	rightHeader rect
	leftBody    rect
	rightBody   rect
}

func newSplitBodyLayout(width, height, leftWidth int) splitBodyLayout {
	if width < 3 {
		width = 3
	}
	if height < 1 {
		height = 1
	}
	leftWidth = clampInt(leftWidth, 1, maxInt(1, width-2))
	rightWidth := maxInt(1, width-leftWidth-1)
	leftWidth = maxInt(1, width-rightWidth-1)
	return splitBodyLayout{
		leftHeader:  rect{x: 0, y: 0, width: leftWidth, height: 1},
		rightHeader: rect{x: leftWidth + 1, y: 0, width: rightWidth, height: 1},
		leftBody:    rect{x: 0, y: 1, width: leftWidth, height: maxInt(0, height-1)},
		rightBody:   rect{x: leftWidth + 1, y: 1, width: rightWidth, height: maxInt(0, height-1)},
	}
}

func renderPanel(title, body string, width, height int) string {
	width = maxInt(8, width)
	height = maxInt(3, height)
	bodyLines := normalizeLines(body, width-2, height-2)
	lines := make([]string, 0, height)
	lines = append(lines, borderStyle.Render(makeTopBorder(title, width)))
	for _, line := range bodyLines {
		lines = append(lines, borderStyle.Render("│")+panelBodyStyle.Render(line)+borderStyle.Render("│"))
	}
	lines = append(lines, borderStyle.Render("╰"+strings.Repeat("─", width-2)+"╯"))
	return strings.Join(lines, "\n")
}

func renderSplitBody(leftTitle, leftBody, rightTitle, rightBody string, width, height, leftWidth int) string {
	layout := newSplitBodyLayout(width, height, leftWidth)
	leftLines := normalizeLines(leftBody, layout.leftBody.width, layout.leftBody.height)
	rightLines := normalizeLines(rightBody, layout.rightBody.width, layout.rightBody.height)
	lines := make([]string, 0, maxInt(1, height))
	header := titleRowStyle.Render(padVisual(leftTitle, layout.leftHeader.width)) + borderStyle.Render("│") + titleRowStyle.Render(padVisual(rightTitle, layout.rightHeader.width))
	lines = append(lines, header)
	for i := 0; i < maxInt(len(leftLines), len(rightLines)); i++ {
		leftLine := ""
		rightLine := ""
		if i < len(leftLines) {
			leftLine = leftLines[i]
		}
		if i < len(rightLines) {
			rightLine = rightLines[i]
		}
		lines = append(lines, panelBodyStyle.Render(leftLine)+borderStyle.Render("│")+panelBodyStyle.Render(rightLine))
	}
	return strings.Join(lines, "\n")
}

func renderMenuBody(items []string, cursor, width, height int, focused bool) (string, []rect) {
	width = maxInt(1, width)
	height = maxInt(1, height)
	lines := make([]string, 0, height)
	itemRects := make([]rect, 0, len(items))
	topPad := 1
	if height <= len(items) {
		topPad = 0
	}
	for i := 0; i < topPad && len(lines) < height; i++ {
		lines = append(lines, padVisual("", width))
	}
	for idx, item := range items {
		text := "  " + item + "  "
		style := navItemStyle
		if idx == cursor {
			text = "> " + item + " <"
			style = navActiveStyle
			if focused {
				style = navFocusStyle
			}
		}
		rowY := len(lines)
		if rowY >= height {
			break
		}
		centered := centerVisual(style.Render(text), width)
		lines = append(lines, centered)
		itemRects = append(itemRects, rect{x: 0, y: rowY, width: width, height: 1})
	}
	for len(lines) < height {
		lines = append(lines, padVisual("", width))
	}
	return strings.Join(lines, "\n"), itemRects
}

func normalizeLines(body string, width, height int) []string {
	width = maxInt(1, width)
	height = maxInt(1, height)
	raw := strings.Split(body, "\n")
	lines := make([]string, 0, height)
	for _, line := range raw {
		if len(lines) >= height {
			break
		}
		lines = append(lines, padVisual(line, width))
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return lines
}

func makeTopBorder(title string, width int) string {
	minTitleWidth := maxInt(1, width-6)
	title = ansi.Truncate(title, minTitleWidth, "")
	segment := "|" + title + "|"
	prefix := "╭─"
	suffix := "─╮"
	remaining := width - lipgloss.Width(prefix) - lipgloss.Width(segment) - lipgloss.Width(suffix)
	if remaining < 0 {
		segment = "|" + ansi.Truncate(title, maxInt(1, width-8), "") + "|"
		remaining = maxInt(0, width-lipgloss.Width(prefix)-lipgloss.Width(segment)-lipgloss.Width(suffix))
	}
	return prefix + titleStyle.Render(segment) + strings.Repeat("─", remaining) + suffix
}

func padVisual(text string, width int) string {
	trimmed := ansi.Truncate(text, maxInt(1, width), "")
	padding := maxInt(0, width-lipgloss.Width(trimmed))
	return trimmed + strings.Repeat(" ", padding)
}

func centerVisual(text string, width int) string {
	trimmed := ansi.Truncate(text, maxInt(1, width), "")
	remaining := maxInt(0, width-lipgloss.Width(trimmed))
	left := remaining / 2
	right := remaining - left
	return strings.Repeat(" ", left) + trimmed + strings.Repeat(" ", right)
}
