package manager

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var errNoImportableMods = errors.New("no importable mods found in the selected zip")

type archiveDirStats struct {
	hasDLL bool
	hasPCK bool
	files  []*zip.File
}

type archiveModScan struct {
	ArchiveDir string
	Name       string
}

type archiveExportSource struct {
	Name string
	Path string
}

func (m *Manager) PreviewZipMods(zipPath string) ([]ArchiveImportCandidate, error) {
	scans, err := scanZipModCandidates(zipPath)
	if err != nil {
		return nil, err
	}
	result := make([]ArchiveImportCandidate, 0, len(scans))
	for _, scan := range scans {
		result = append(result, ArchiveImportCandidate{Name: scan.Name})
	}
	return result, nil
}

func (m *Manager) ImportAvailableModsFromZip(zipPath string) ([]ArchiveImportResult, error) {
	return m.importZipMods(zipPath, m.PreferredAvailableModsRoot(), false)
}

func (m *Manager) ImportInstalledModsFromZip(gameDir, zipPath string) ([]ArchiveImportResult, error) {
	return m.importZipMods(zipPath, filepath.Join(gameDir, "mods"), true)
}

func (m *Manager) importZipMods(zipPath, destRoot string, enableMods bool) ([]ArchiveImportResult, error) {
	if err := ensureDir(destRoot); err != nil {
		return nil, err
	}
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	scans, err := scanZipReaderModCandidates(&reader.Reader, zipPath)
	if err != nil {
		return nil, err
	}
	results := make([]ArchiveImportResult, 0, len(scans))
	for _, scan := range scans {
		destDir := filepath.Join(destRoot, scan.Name)
		result := ArchiveImportResult{Name: scan.Name, Destination: destDir}
		for _, file := range filesUnderArchiveDir(reader.File, scan.ArchiveDir) {
			relPath, ok := archiveRelativePath(file.Name, scan.ArchiveDir)
			if !ok || relPath == "" {
				continue
			}
			targetPath, err := safeJoinWithinRoot(destDir, relPath)
			if err != nil {
				return nil, err
			}
			if file.FileInfo().IsDir() {
				if err := ensureDir(targetPath); err != nil {
					return nil, err
				}
				continue
			}
			if err := ensureDir(filepath.Dir(targetPath)); err != nil {
				return nil, err
			}
			_, _, err = copyStreamWithReplaceFallback(file.Open, targetPath, file.Mode(), archiveFileModified(file))
			if err != nil {
				return nil, err
			}
			result.FilesCopied++
		}
		if enableMods {
			changed, err := m.EnableModsInSettings()
			if err == nil {
				result.EnableChanged = changed
			}
		}
		results = append(results, result)
	}
	return results, nil
}

func (m *Manager) ExportAvailableModsToZip(mods []ModPackage, zipPath string) (ArchiveExportResult, error) {
	sources := make([]archiveExportSource, 0, len(mods))
	for _, mod := range mods {
		sources = append(sources, archiveExportSource{Name: mod.DirName, Path: mod.SourcePath})
	}
	return exportModFoldersToZip(sources, zipPath)
}

func (m *Manager) ExportInstalledModsToZip(gameDir string, names []string, zipPath string) (ArchiveExportResult, error) {
	sources := make([]archiveExportSource, 0, len(names))
	modsDir := filepath.Join(gameDir, "mods")
	for _, name := range names {
		sources = append(sources, archiveExportSource{Name: name, Path: filepath.Join(modsDir, name)})
	}
	return exportModFoldersToZip(sources, zipPath)
}

func exportModFoldersToZip(sources []archiveExportSource, zipPath string) (ArchiveExportResult, error) {
	if len(sources) == 0 {
		return ArchiveExportResult{}, fmt.Errorf("no mods selected for export")
	}
	if filepath.Ext(zipPath) == "" {
		zipPath += ".zip"
	}
	if err := ensureDir(filepath.Dir(zipPath)); err != nil {
		return ArchiveExportResult{}, err
	}
	file, err := os.Create(zipPath)
	if err != nil {
		return ArchiveExportResult{}, err
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	filesAdded := 0
	for _, source := range sources {
		if err := filepath.WalkDir(source.Path, func(current string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			relPath, err := filepath.Rel(source.Path, current)
			if err != nil {
				return err
			}
			archivePath := path.Join(source.Name, filepath.ToSlash(relPath))
			info, err := d.Info()
			if err != nil {
				return err
			}
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}
			header.Name = archivePath
			header.Method = zip.Deflate
			entryWriter, err := writer.CreateHeader(header)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(current)
			if err != nil {
				return err
			}
			if _, err := entryWriter.Write(data); err != nil {
				return err
			}
			filesAdded++
			return nil
		}); err != nil {
			_ = writer.Close()
			return ArchiveExportResult{}, err
		}
	}
	if err := writer.Close(); err != nil {
		return ArchiveExportResult{}, err
	}
	return ArchiveExportResult{ZipPath: zipPath, ModCount: len(sources), FilesAdded: filesAdded}, nil
}

func scanZipModCandidates(zipPath string) ([]archiveModScan, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return scanZipReaderModCandidates(&reader.Reader, zipPath)
}

func scanZipReaderModCandidates(reader *zip.Reader, zipPath string) ([]archiveModScan, error) {
	byDir := map[string]*archiveDirStats{}
	for _, file := range reader.File {
		normalized, ok := normalizeArchiveName(file.Name)
		if !ok {
			continue
		}
		dir := path.Dir(normalized)
		if dir == "." {
			dir = ""
		}
		stats := byDir[dir]
		if stats == nil {
			stats = &archiveDirStats{}
			byDir[dir] = stats
		}
		if !file.FileInfo().IsDir() {
			stats.files = append(stats.files, file)
			switch strings.ToLower(path.Ext(normalized)) {
			case ".dll":
				stats.hasDLL = true
			case ".pck":
				stats.hasPCK = true
			}
		}
	}

	rootStats := byDir[""]
	candidateDirs := make([]string, 0)
	if rootStats != nil && rootStats.hasDLL && rootStats.hasPCK {
		candidateDirs = append(candidateDirs, "")
	} else {
		for dir, stats := range byDir {
			if dir == "" || !stats.hasDLL || !stats.hasPCK {
				continue
			}
			candidateDirs = append(candidateDirs, dir)
		}
	}
	if len(candidateDirs) == 0 {
		return nil, errNoImportableMods
	}
	sort.Strings(candidateDirs)
	scans := make([]archiveModScan, 0, len(candidateDirs))
	for _, dir := range candidateDirs {
		stats := byDir[dir]
		fallbackName := archiveFallbackName(zipPath, dir)
		name := archiveModName(stats.files, fallbackName)
		scans = append(scans, archiveModScan{ArchiveDir: dir, Name: name})
	}
	return scans, nil
}

func archiveModName(files []*zip.File, fallback string) string {
	manifest := archiveManifest(files, fallback)
	return installNameFromFileNames(fallback, archiveCandidateFileNames(files), manifest)
}

func archiveManifest(files []*zip.File, fallback string) *ModManifest {
	allowed := archiveManifestNames(files, fallback)
	jsonFiles := make([]*zip.File, 0)
	for _, file := range files {
		if file.FileInfo().IsDir() || strings.ToLower(path.Ext(file.Name)) != ".json" {
			continue
		}
		if !allowed[strings.ToLower(path.Base(file.Name))] {
			continue
		}
		jsonFiles = append(jsonFiles, file)
	}
	sort.SliceStable(jsonFiles, func(i, j int) bool {
		left := strings.ToLower(path.Base(jsonFiles[i].Name))
		right := strings.ToLower(path.Base(jsonFiles[j].Name))
		if left == "mod_manifest.json" {
			return false
		}
		if right == "mod_manifest.json" {
			return true
		}
		return left < right
	})
	for _, file := range jsonFiles {
		reader, err := file.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			continue
		}
		manifest := &ModManifest{}
		if err := json.Unmarshal(stripUTF8BOM(data), manifest); err == nil {
			return manifest
		}
	}
	return nil
}

func archiveManifestNames(files []*zip.File, fallback string) map[string]bool {
	allowed := map[string]bool{
		"mod_manifest.json":                 true,
		strings.ToLower(fallback + ".json"): true,
	}
	for _, name := range archiveCandidateFileNames(files) {
		base := strings.TrimSuffix(name, filepath.Ext(name))
		if base == "" {
			continue
		}
		allowed[strings.ToLower(base+".json")] = true
	}
	return allowed
}

func archiveCandidateFileNames(files []*zip.File) []string {
	names := make([]string, 0, len(files))
	for _, file := range files {
		if file.FileInfo().IsDir() {
			continue
		}
		names = append(names, path.Base(file.Name))
	}
	return names
}

func archiveFallbackName(zipPath, archiveDir string) string {
	if archiveDir == "" {
		return strings.TrimSuffix(filepath.Base(zipPath), filepath.Ext(zipPath))
	}
	return path.Base(archiveDir)
}

func filesUnderArchiveDir(files []*zip.File, archiveDir string) []*zip.File {
	result := make([]*zip.File, 0)
	for _, file := range files {
		if _, ok := archiveRelativePath(file.Name, archiveDir); ok {
			result = append(result, file)
		}
	}
	return result
}

func archiveRelativePath(name, archiveDir string) (string, bool) {
	normalized, ok := normalizeArchiveName(name)
	if !ok {
		return "", false
	}
	if archiveDir == "" {
		return normalized, true
	}
	prefix := archiveDir + "/"
	if normalized == archiveDir {
		return "", true
	}
	if !strings.HasPrefix(normalized, prefix) {
		return "", false
	}
	return strings.TrimPrefix(normalized, prefix), true
}

func normalizeArchiveName(name string) (string, bool) {
	normalized := strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	normalized = strings.TrimPrefix(normalized, "/")
	if normalized == "" {
		return "", false
	}
	clean := path.Clean(normalized)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	return clean, true
}

func safeJoinWithinRoot(root, rel string) (string, error) {
	cleanRel := filepath.Clean(filepath.FromSlash(rel))
	if cleanRel == "." || strings.HasPrefix(cleanRel, "..") {
		return "", fmt.Errorf("invalid archive path: %s", rel)
	}
	target := filepath.Join(root, cleanRel)
	cleanRoot := filepath.Clean(root)
	if target != cleanRoot && !strings.HasPrefix(target, cleanRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("archive path escaped root: %s", rel)
	}
	return target, nil
}

func archiveFileModified(file *zip.File) time.Time {
	if !file.Modified.IsZero() {
		return file.Modified
	}
	return time.Now()
}
