package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
)

type logEntry struct {
	stamp string
	level string
	text  string
}

type logModel struct {
	entries    []logEntry
	viewport   viewport.Model
	maxEntries int
}

func newLogModel() logModel {
	return logModel{
		viewport:   viewport.New(0, 0),
		maxEntries: 200,
	}
}

func (l *logModel) Add(level, text string) {
	l.entries = append(l.entries, logEntry{stamp: time.Now().Format("15:04:05"), level: level, text: text})
	if len(l.entries) > l.maxEntries {
		l.entries = l.entries[len(l.entries)-l.maxEntries:]
	}
	l.sync()
	l.viewport.GotoBottom()
}

func (l *logModel) Resize(width, height int) {
	l.viewport.Width = maxInt(1, width)
	l.viewport.Height = maxInt(1, height)
	l.sync()
	l.viewport.GotoBottom()
}

func (l *logModel) View() string {
	return l.viewport.View()
}

func (l *logModel) sync() {
	lines := make([]string, 0, len(l.entries))
	for _, entry := range l.entries {
		label := entry.level
		switch entry.level {
		case "ok":
			label = okStyle.Render("OK")
		case "warn":
			label = warnStyle.Render("WARN")
		case "error":
			label = errorStyle.Render("ERR")
		default:
			label = mutedStyle.Render("INFO")
		}
		lines = append(lines, fmt.Sprintf("%s  [%s] %s", mutedStyle.Render(entry.stamp), label, entry.text))
	}
	l.viewport.SetContent(strings.Join(lines, "\n"))
}
