package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"slaymodgo/internal/manager"
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
	status := ""
	switch {
	case mod.Updatable:
		status = t(" (update available locally)")
	case mod.Installed:
		status = t(" (installed)")
	}
	return fmt.Sprintf("%s %s%s", checkbox, mod.Label, status)
}

func renderInstalledModListLabel(mod manager.InstalledMod, selected bool) string {
	checkbox := "[ ]"
	if selected {
		checkbox = "[x]"
	}
	return fmt.Sprintf("%s %s", checkbox, mod.Label)
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
	if width < 1 {
		width = 1
	}
	if height < 2 {
		height = 2
	}
	header := sectionTitleStyle.Width(width).Render(title)
	divider := sectionDividerStyle.Width(width).Render(strings.Repeat("-", width))
	bodyHeight := maxInt(1, height-2)
	bodyView := sectionBodyStyle.Width(width).Height(bodyHeight).Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, header, divider, bodyView)
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
	leftBox := lipgloss.NewStyle().Width(leftWidth).Render(left)
	gap := lipgloss.NewStyle().Width(2).Render("")
	rightBox := lipgloss.NewStyle().Width(rightWidth).Render(right)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, gap, rightBox)
}

func renderList(items []string, cursor int, focused bool) string {
	var lines []string
	for idx, item := range items {
		prefix := "  "
		style := lipgloss.NewStyle()
		if idx == cursor {
			prefix = "> "
			style = cursorStyle
			if focused {
				style = focusStyle
			}
		}
		lines = append(lines, style.Render(prefix+item))
	}
	if len(lines) == 0 {
		return mutedStyle.Render(t("(empty)"))
	}
	return strings.Join(lines, "\n")
}
