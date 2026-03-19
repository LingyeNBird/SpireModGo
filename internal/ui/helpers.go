package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"spiremodgo/internal/manager"
)

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Local().Format("2006-01-02 15:04")
}

func formatSaveTypeName(saveType manager.SaveType) string {
	if saveType == manager.SaveTypeNormal {
		return t("Vanilla")
	}
	return t("Modded")
}

func formatSaveRef(saveType manager.SaveType, slot int) string {
	prefix := "A"
	if saveType == manager.SaveTypeModded {
		prefix = "B"
	}
	return fmt.Sprintf("%s%d", prefix, slot)
}

func renderModListLabel(mod manager.ModPackage, selected bool) string {
	checkbox := "[ ]"
	if selected {
		checkbox = "[x]"
	}
	badge := ""
	if mod.NeedsRepair {
		badge = oldFormatBadgeStyle.Render("[?]") + " "
	}
	status := ""
	switch {
	case mod.Updatable:
		status = t(" (update available locally)")
	case mod.Installed:
		status = t(" (installed)")
	}
	return fmt.Sprintf("%s %s%s%s", checkbox, badge, mod.Label, status)
}

func renderModListEntry(mod manager.ModPackage, checked, selected, focused bool) string {
	checkbox := "[ ]"
	if checked {
		checkbox = "[x]"
	}
	status := ""
	switch {
	case mod.Updatable:
		status = t(" (update available locally)")
	case mod.Installed:
		status = t(" (installed)")
	}
	return renderModEntryLine(checkbox, mod.Label+status, mod.NeedsRepair, selected, focused)
}

func renderInstalledModListLabel(mod manager.InstalledMod, selected bool) string {
	checkbox := "[ ]"
	if selected {
		checkbox = "[x]"
	}
	badge := ""
	if mod.NeedsRepair {
		badge = oldFormatBadgeStyle.Render("[?]") + " "
	}
	return fmt.Sprintf("%s %s%s", checkbox, badge, mod.Label)
}

func renderInstalledModListEntry(mod manager.InstalledMod, checked, selected, focused bool) string {
	checkbox := "[ ]"
	if checked {
		checkbox = "[x]"
	}
	return renderModEntryLine(checkbox, mod.Label, mod.NeedsRepair, selected, focused)
}

func renderModEntryLine(checkbox, label string, needsRepair, selected, focused bool) string {
	prefix := "  "
	lineStyle := lipgloss.NewStyle()
	if selected {
		prefix = "> "
		lineStyle = cursorStyle
		if focused {
			lineStyle = focusStyle
		}
	}
	badge := ""
	if needsRepair {
		badge = oldFormatBadgeStyle.Render("[") + oldFormatBadgeStyle.Render("?") + oldFormatBadgeStyle.Render("]") + " "
	}
	if !selected {
		return prefix + checkbox + " " + badge + label
	}
	return lineStyle.Render(prefix+checkbox+" ") + badge + lineStyle.Render(label)
}

func renderAvailableModDetail(mod manager.ModPackage) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%s\n\n", mod.Label)
	fmt.Fprintf(&builder, t("Install folder: %s\n"), mod.InstallName)
	fmt.Fprintf(&builder, t("Source path: %s\n"), mod.SourcePath)

	status := t("Not installed")
	switch {
	case mod.Updatable:
		status = t("Installed v%s, package v%s", mod.InstalledVersion, manifestVersion(mod.Manifest))
	case mod.Installed:
		if mod.InstalledVersion != "" {
			status = t("Installed v%s", mod.InstalledVersion)
		} else {
			status = t("Installed")
		}
	}
	fmt.Fprintf(&builder, t("Status: %s\n"), status)
	if mod.NeedsRepair {
		builder.WriteString(t("Repair status: %s", t("Old format suspected")) + "\n")
	}

	if mod.Manifest == nil {
		builder.WriteString("\n" + t("No mod_manifest.json detected. The install folder name is inferred from files.\n"))
		return builder.String()
	}

	builder.WriteString("\n" + t("Manifest") + "\n")
	if mod.Manifest.Name != "" {
		fmt.Fprintf(&builder, t("Name: %s\n"), mod.Manifest.Name)
	}
	if mod.Manifest.Version != "" {
		fmt.Fprintf(&builder, t("Version: %s\n"), mod.Manifest.Version)
	}
	if mod.Manifest.Author != "" {
		fmt.Fprintf(&builder, t("Author: %s\n"), mod.Manifest.Author)
	}
	if mod.Manifest.PckName != "" {
		fmt.Fprintf(&builder, t("Package key: %s\n"), mod.Manifest.PckName)
	}
	return builder.String()
}

func renderInstalledModDetail(mod manager.InstalledMod) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%s\n\n", mod.Label)
	fmt.Fprintf(&builder, t("Install path: %s\n"), mod.FullPath)
	if mod.NeedsRepair {
		builder.WriteString(t("Repair status: %s", t("Old format suspected")) + "\n")
	}

	if mod.Manifest != nil {
		builder.WriteString("\n" + t("Manifest") + "\n")
		if mod.Manifest.Name != "" {
			fmt.Fprintf(&builder, t("Name: %s\n"), mod.Manifest.Name)
		}
		if mod.Manifest.Version != "" {
			fmt.Fprintf(&builder, t("Version: %s\n"), mod.Manifest.Version)
		}
		if mod.Manifest.Author != "" {
			fmt.Fprintf(&builder, t("Author: %s\n"), mod.Manifest.Author)
		}
	}

	entries, err := os.ReadDir(mod.FullPath)
	if err != nil {
		fmt.Fprintf(&builder, "\n"+t("Failed to read files: %s\n"), err.Error())
		return builder.String()
	}

	builder.WriteString("\n" + t("Files") + "\n")
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			fmt.Fprintf(&builder, "- %s/\n", entry.Name())
			continue
		}
		info, err := entry.Info()
		if err != nil {
			fmt.Fprintf(&builder, "- %s\n", entry.Name())
			count++
			continue
		}
		fmt.Fprintf(&builder, "- %s (%s)\n", entry.Name(), humanSize(info.Size()))
		count++
	}
	if count == 0 {
		builder.WriteString(t("(no files found)\n"))
	}
	return builder.String()
}

func renderBackupDetail(backup manager.BackupInfo) string {
	return t("%s\n\nType: %s\nSlot: %d\nFiles: %d\nPath: %s\n",
		backup.Name,
		formatSaveTypeName(backup.Type),
		backup.Slot,
		backup.FileCount,
		backup.FullPath,
	)
}

func buildSlotStatus(info manager.SaveSlotInfo) string {
	if !info.HasData {
		return t("(empty)")
	}
	parts := []string{formatTimestamp(info.LastModified)}
	if info.HasCurrentRun {
		parts = append(parts, t("current run"))
	}
	return strings.Join(parts, " | ")
}

func humanSize(size int64) string {
	switch {
	case size >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(size)/(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(size)/(1<<10))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func manifestVersion(manifest *manager.ModManifest) string {
	if manifest == nil || manifest.Version == "" {
		return t("unknown")
	}
	return manifest.Version
}

func dirFileCount(path string) int {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			count++
		}
	}
	return count
}

func countBakFiles(root string) int {
	count := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.Contains(strings.ToLower(d.Name()), ".bak") {
			count++
		}
		return nil
	})
	return count
}

func renderFlatSection(title, body string, width, height int) string {
	return renderPanel(title, body, width, height)
}

func renderFlatColumn(title, body string, width, height int) string {
	return renderFlatSection(title, body, width, height)
}

func splitContentWidths(totalWidth, minLeft, minRight int) (int, int) {
	const gap = 2
	usableWidth := maxInt(2, totalWidth-gap)
	leftWidth := usableWidth / 2
	rightWidth := usableWidth - leftWidth
	if usableWidth >= minLeft+minRight {
		leftWidth = minLeft
		rightWidth = usableWidth - leftWidth
		if rightWidth < minRight {
			rightWidth = minRight
			leftWidth = usableWidth - rightWidth
		}
	}
	leftWidth = maxInt(1, leftWidth)
	rightWidth = maxInt(1, rightWidth)
	return leftWidth, rightWidth
}

func joinFlatColumns(left, right string, leftWidth, rightWidth int) string {
	return joinColumns(left, right, leftWidth, rightWidth)
}

func renderList(items []string, cursor int, focused bool) string {
	var lines []string
	for idx, item := range items {
		lines = append(lines, renderSelectableLine(item, idx == cursor, focused))
	}
	if len(lines) == 0 {
		return mutedStyle.Render(t("(empty)"))
	}
	return strings.Join(lines, "\n")
}

func formatButtonLabel(label string) string {
	return "[ " + label + " ]"
}

func formatSelectorLabel(value string) string {
	return formatButtonLabel("< " + value + " >")
}

func renderInlineButton(label string, active, focused bool) string {
	text := formatButtonLabel(label)
	if active {
		if focused {
			return buttonFocusStyle.Render(text)
		}
		return buttonActiveStyle.Render(text)
	}
	return buttonStyle.Render(text)
}

func renderInlineButtonGroup(labels []string, active int, focused bool) string {
	rendered := make([]string, 0, len(labels))
	for idx, label := range labels {
		rendered = append(rendered, renderInlineButton(label, idx == active, focused && idx == active))
	}
	return strings.Join(rendered, " ")
}

func inlineButtonIndexAt(labels []string, localX int) int {
	offset := 0
	for idx, label := range labels {
		width := lipgloss.Width(formatButtonLabel(label))
		if localX >= offset && localX < offset+width {
			return idx
		}
		offset += width
		if idx < len(labels)-1 {
			offset++
		}
	}
	return -1
}

func renderActionButtonList(labels []string, active int, focused bool) string {
	lines := make([]string, 0, len(labels))
	for idx, label := range labels {
		lines = append(lines, renderInlineButton(label, idx == active, focused && idx == active))
	}
	return strings.Join(lines, "\n")
}

func renderFooterSegment(label string, active bool, activeStyle, inactiveStyle lipgloss.Style) string {
	text := "|" + label + "|"
	if active {
		return activeStyle.Render(text)
	}
	return inactiveStyle.Render(text)
}

func renderSelectableLine(text string, selected, focused bool) string {
	prefix := "  "
	style := lipgloss.NewStyle()
	if selected {
		prefix = "> "
		style = cursorStyle
		if focused {
			style = focusStyle
		}
	}
	return style.Render(prefix + text)
}

func renderActionLine(label string, active bool) string {
	return renderInlineButton(label, active, false)
}

func renderValueControl(label, value string) string {
	return label + "  " + formatSelectorLabel(value)
}

func renderValueControlWithDetail(label, value, detail string, selected, focused bool) string {
	lines := []string{renderSelectableLine(renderValueControl(label, value), selected, focused)}
	if strings.TrimSpace(detail) != "" {
		indent := strings.Repeat(" ", lipgloss.Width(label)+4)
		lines = append(lines, mutedStyle.Render(indent+"<"+detail+">"))
	}
	return strings.Join(lines, "\n")
}

func padColumnText(text string, width int) string {
	return text + strings.Repeat(" ", maxInt(0, width-lipgloss.Width(text)))
}
