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
	if width < 5 {
		width = 5
	}
	if height < 3 {
		height = 3
	}
	bodyWidth := maxInt(2, width-3)
	bodyHeight := maxInt(1, height-2)
	leftWidth = clampInt(leftWidth, 1, maxInt(1, bodyWidth-1))
	rightWidth := maxInt(1, bodyWidth-leftWidth)
	leftWidth = maxInt(1, bodyWidth-rightWidth)
	return splitBodyLayout{
		leftHeader:  rect{x: 1, y: 0, width: leftWidth, height: 1},
		rightHeader: rect{x: leftWidth + 2, y: 0, width: rightWidth, height: 1},
		leftBody:    rect{x: 1, y: 1, width: leftWidth, height: bodyHeight},
		rightBody:   rect{x: leftWidth + 2, y: 1, width: rightWidth, height: bodyHeight},
	}
}

func renderPanel(title, body string, width, height int) string {
	width = maxInt(8, width)
	height = maxInt(3, height)
	bodyLines := normalizeLines(body, width-2, height-2)
	lines := make([]string, 0, height)
	lines = append(lines, makeTopBorder(title, width))
	for _, line := range bodyLines {
		lines = append(lines, borderStyle.Render("│")+panelBodyStyle.Render(line)+borderStyle.Render("│"))
	}
	lines = append(lines, borderStyle.Render("╰"+strings.Repeat("─", width-2)+"╯"))
	return strings.Join(lines, "\n")
}

func renderSplitBody(leftTitle, leftBody, rightTitle, rightBody string, width, height, leftWidth int) string {
	return renderSplitPanel(leftTitle, leftBody, rightTitle, rightBody, width, height, leftWidth)
}

func renderSplitPanel(leftTitle, leftBody, rightTitle, rightBody string, width, height, leftWidth int) string {
	layout := newSplitBodyLayout(width, height, leftWidth)
	leftLines := normalizeLines(leftBody, layout.leftBody.width, layout.leftBody.height)
	rightLines := normalizeLines(rightBody, layout.rightBody.width, layout.rightBody.height)
	lines := make([]string, 0, maxInt(3, height))
	lines = append(lines, makeSplitTopBorder(leftTitle, rightTitle, layout.leftBody.width, layout.rightBody.width))
	for i := 0; i < maxInt(len(leftLines), len(rightLines)); i++ {
		leftLine := ""
		rightLine := ""
		if i < len(leftLines) {
			leftLine = leftLines[i]
		}
		if i < len(rightLines) {
			rightLine = rightLines[i]
		}
		lines = append(lines, borderStyle.Render("│")+panelBodyStyle.Render(leftLine)+borderStyle.Render("│")+panelBodyStyle.Render(rightLine)+borderStyle.Render("│"))
	}
	lines = append(lines, borderStyle.Render("╰"+strings.Repeat("─", layout.leftBody.width)+"┴"+strings.Repeat("─", layout.rightBody.width)+"╯"))
	return strings.Join(lines, "\n")
}

func renderBorderButtonRow(labels []string, width int, active int) string {
	return renderBorderButtonRowStyled(labels, width, active, borderStyle, titleStyle, mutedStyle)
}

func renderBorderButtonRowStyled(labels []string, width int, active int, border lipgloss.Style, activeTitle lipgloss.Style, inactiveTitle lipgloss.Style) string {
	segments := make([]string, 0, len(labels))
	for idx, label := range labels {
		segments = append(segments, renderFooterSegment(label, idx == active, activeTitle, inactiveTitle))
	}
	bodyWidth := maxInt(1, width-2)
	line := "╰"
	used := 1
	for idx, segment := range segments {
		prefix := "─"
		if idx == 0 {
			prefix = "───"
		}
		line += prefix + segment
		used += lipgloss.Width(prefix) + lipgloss.Width(segment)
	}
	if bodyWidth+1-used > 0 {
		line += strings.Repeat("─", bodyWidth+1-used)
	}
	line += "╯"
	return border.Render(line)
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
	return makeTopBorderStyled(title, width, borderStyle, titleStyle)
}

func makeTopBorderStyled(title string, width int, border lipgloss.Style, titleRenderer lipgloss.Style) string {
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
	return border.Render(prefix) + titleRenderer.Render(segment) + border.Render(strings.Repeat("─", remaining)+suffix)
}

func makeSplitTopBorder(leftTitle, rightTitle string, leftWidth, rightWidth int) string {
	leftSpan := makeTitledBorderSpan(leftTitle, leftWidth, 1)
	rightSpan := makeTitledBorderSpan(rightTitle, rightWidth, 3)
	return borderStyle.Render("╭") + leftSpan + borderStyle.Render("┬") + rightSpan + borderStyle.Render("╮")
}

func makeTitledBorderSpan(title string, width, leadingDashes int) string {
	width = maxInt(1, width)
	leading := minInt(leadingDashes, maxInt(0, width-3))
	maxTitleWidth := maxInt(1, width-leading-2)
	segment := "|" + ansi.Truncate(title, maxTitleWidth, "") + "|"
	segmentWidth := lipgloss.Width(segment)
	if leading+segmentWidth > width {
		leading = maxInt(0, width-segmentWidth)
	}
	trailing := maxInt(0, width-leading-segmentWidth)
	return borderStyle.Render(strings.Repeat("─", leading)) + titleStyle.Render(segment) + borderStyle.Render(strings.Repeat("─", trailing))
}

func padVisual(text string, width int) string {
	trimmed := ansi.Truncate(text, maxInt(1, width), "")
	padding := maxInt(0, width-lipgloss.Width(trimmed))
	return trimmed + strings.Repeat(" ", padding)
}

func wrapBodyText(body string, width int) string {
	width = maxInt(1, width)
	raw := strings.Split(body, "\n")
	wrapped := make([]string, 0, len(raw))
	for _, line := range raw {
		if line == "" {
			wrapped = append(wrapped, "")
			continue
		}
		wrapped = append(wrapped, strings.Split(ansi.Hardwrap(line, width, true), "\n")...)
	}
	return strings.Join(wrapped, "\n")
}

func centerVisual(text string, width int) string {
	trimmed := ansi.Truncate(text, maxInt(1, width), "")
	remaining := maxInt(0, width-lipgloss.Width(trimmed))
	left := remaining / 2
	right := remaining - left
	return strings.Repeat(" ", left) + trimmed + strings.Repeat(" ", right)
}
