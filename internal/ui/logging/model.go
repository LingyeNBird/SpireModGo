package logging

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type entry struct {
	stamp string
	level string
	key   string
	args  []any
}

type Formatter func(stamp, level, key string, args ...any) string

type Model struct {
	entries    []entry
	viewport   viewport.Model
	maxEntries int
	formatter  Formatter
}

func New(formatter Formatter) Model {
	return Model{
		viewport:   viewport.New(0, 0),
		maxEntries: 200,
		formatter:  formatter,
	}
}

func (l *Model) Add(level, key string, args ...any) {
	l.entries = append(l.entries, entry{stamp: time.Now().Format("15:04:05"), level: level, key: key, args: args})
	if len(l.entries) > l.maxEntries {
		l.entries = l.entries[len(l.entries)-l.maxEntries:]
	}
	l.sync()
	l.viewport.GotoBottom()
}

func (l *Model) Resize(width, height int) {
	width = max(1, width)
	height = max(1, height)
	if l.viewport.Width == width && l.viewport.Height == height {
		return
	}
	l.viewport.Width = width
	l.viewport.Height = height
	l.sync()
}

func (l *Model) View() string {
	return l.viewport.View()
}

func (l *Model) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	l.viewport, cmd = l.viewport.Update(msg)
	return cmd
}

func (l *Model) Sync() {
	l.sync()
}

func (l *Model) sync() {
	lines := make([]string, 0, len(l.entries))
	for _, entry := range l.entries {
		lines = append(lines, l.formatter(entry.stamp, entry.level, entry.key, entry.args...))
	}
	l.viewport.SetContent(strings.Join(lines, "\n"))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
