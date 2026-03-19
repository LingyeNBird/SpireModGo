package ui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"slaymodgo/internal/manager"
)

func loadTestLocalizer(tt *testing.T) {
	tt.Helper()
	if err := loadLocalizer(filepath.Clean(filepath.Join("..", ".."))); err != nil {
		tt.Fatalf("load localizer: %v", err)
	}
}

func TestWrapBodyTextWrapsWithoutDroppingContent(tt *testing.T) {
	text := "当前路径与状态当前路径与状态"
	got := wrapBodyText(text, 8)
	if !strings.Contains(got, "\n") {
		tt.Fatalf("expected wrapped text to contain a newline, got %q", got)
	}
	if strings.ReplaceAll(got, "\n", "") != text {
		tt.Fatalf("expected wrapped text to preserve content, got %q", got)
	}
}

func TestOverlayModalPreservesBackgroundOutsideModal(tt *testing.T) {
	base := strings.Join([]string{"0123456789", "abcdefghij"}, "\n")
	overlay := strings.Join([]string{"   ╭─╮    ", "   │X│    "}, "\n")
	got := overlayModal(base, overlay, 10, 2)
	lines := strings.Split(got, "\n")
	if lines[0] != "012╭─╮6789" {
		tt.Fatalf("expected top line to preserve background around modal, got %q", lines[0])
	}
	if lines[1] != "abc│X│ghij" {
		tt.Fatalf("expected body line to preserve background around modal, got %q", lines[1])
	}
}

func TestRenderSaveSlotTableAlignsBackupCounts(tt *testing.T) {
	loadTestLocalizer(tt)
	screen := savesScreen{backups: make([]manager.BackupInfo, 0, 17)}
	for range 12 {
		screen.backups = append(screen.backups, manager.BackupInfo{Type: manager.SaveTypeNormal, Slot: 1})
	}
	for range 5 {
		screen.backups = append(screen.backups, manager.BackupInfo{Type: manager.SaveTypeNormal, Slot: 3})
	}
	slots := []manager.SaveSlotInfo{
		{HasData: true, LastModified: time.Date(2026, 3, 18, 23, 42, 0, 0, time.UTC)},
		{HasData: false},
		{HasData: false},
	}
	view := ansi.Strip(screen.renderSaveSlotTable(manager.SaveTypeNormal, slots, 36, 1, true))
	lines := strings.Split(view, "\n")
	if len(lines) != 3 {
		tt.Fatalf("expected three save rows, got %d", len(lines))
	}
	labels := []string{t("%d backups", 12), t("%d backups", 0), t("%d backups", 5)}
	positions := make([]int, len(lines))
	for idx, line := range lines {
		byteIndex := strings.Index(line, labels[idx])
		if byteIndex < 0 {
			tt.Fatalf("expected backup label %q in line %q", labels[idx], line)
		}
		positions[idx] = lipgloss.Width(line[:byteIndex])
	}
	if positions[0] != positions[1] || positions[1] != positions[2] {
		tt.Fatalf("expected aligned backup columns, got positions %v in %q", positions, lines)
	}
	if !strings.HasPrefix(lines[0], ">") {
		tt.Fatalf("expected selected row to keep a visible cursor, got %q", lines[0])
	}
}

func TestModsActionIndexAtUsesRenderedLabelWidths(tt *testing.T) {
	loadTestLocalizer(tt)
	screen := modsScreen{tab: modsTabAvailable}
	labels := screen.actionLabels()
	if got := screen.actionIndexAt(lipgloss.Width(labels[0]) / 2); got != 0 {
		tt.Fatalf("expected select-all hit to map to index 0, got %d", got)
	}
	cancelX := lipgloss.Width(labels[0]) + 2 + lipgloss.Width(labels[1])/2
	if got := screen.actionIndexAt(cancelX); got != 1 {
		tt.Fatalf("expected cancel hit to map to index 1, got %d", got)
	}
	installX := lipgloss.Width(labels[0]) + 2 + lipgloss.Width(labels[1]) + 2 + lipgloss.Width(labels[2])/2
	if got := screen.actionIndexAt(installX); got != 2 {
		tt.Fatalf("expected install hit to map to index 2, got %d", got)
	}
	if got := screen.actionIndexAt(lipgloss.Width(labels[0])); got != -1 {
		tt.Fatalf("expected separator gap to be non-clickable, got %d", got)
	}
}

func TestModsTabIndexAtUsesRenderedTabWidths(tt *testing.T) {
	loadTestLocalizer(tt)
	screen := modsScreen{tab: modsTabAvailable}
	labels := screen.tabLabels()
	availableWidth := lipgloss.Width(">" + labels[0] + "<")
	installedStart := availableWidth + 2
	installedCenter := installedStart + lipgloss.Width(labels[1])/2
	if got := screen.tabIndexAt(installedCenter); got != 1 {
		tt.Fatalf("expected installed-tab hit to map to index 1, got %d", got)
	}
}

func TestNewTranslationKeysResolve(tt *testing.T) {
	loadTestLocalizer(tt)
	checks := map[string]string{
		t("Installed [%d]", 0):        "已安装[0]",
		t("Type: %s", t("Vanilla")):   "类型：原版",
		t("Slot: %d", 1):              "槽位：1",
		t("Status: %s", t("(empty)")): "状态：（空）",
		t("Save Actions"):             "存档操作",
		t("Steam profiles: %d", 1):    "Steam 档案：1",
		t("%d backups", 0):            "0个备份",
	}
	for got, want := range checks {
		if got != want {
			tt.Fatalf("expected translation %q, got %q", want, got)
		}
	}
}

func TestRenderCopyTargetModalUsesOrangeBorderStyle(tt *testing.T) {
	loadTestLocalizer(tt)
	modal := modalState{
		open:  true,
		title: t("Copy Options"),
		kind:  modalKindCopyTarget,
		copyOptions: []copyTargetOption{
			{Header: true, Label: t("Vanilla Saves")},
			{Label: "2", SaveType: manager.SaveTypeNormal, Slot: 2, Status: t("(empty)")},
		},
		optionCursor: 1,
	}
	view := renderCopyTargetModal(60, 20, modal)
	if !strings.Contains(view, copyModalBorderStyle.Render("│")) {
		tt.Fatalf("expected copy modal body border to use orange style")
	}
	if !strings.Contains(view, copyModalBorderStyle.Render("╰")) {
		tt.Fatalf("expected copy modal footer border to use orange style")
	}
}
