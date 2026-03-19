package ui

import (
	"errors"
	"fmt"
	"path/filepath"

	"spiremodgo/internal/manager"
)

type State struct {
	manager         *manager.Manager
	selectedSteamID string
}

func NewState(mgr *manager.Manager) *State {
	return &State{manager: mgr}
}

func (s *State) Manager() *manager.Manager {
	return s.manager
}

func (s *State) GameDir() string {
	return s.manager.GetGameDir()
}

func (s *State) EnsureGameDir() (string, error) {
	dir, err := s.manager.EnsureGameDir()
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", errors.New("game directory is not configured; open Settings to auto-detect or set it manually")
	}
	return dir, nil
}

func (s *State) AutoDetectGameDir() (string, error) {
	dir, err := s.manager.FindGameDir()
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", errors.New("could not find SlayTheSpire2.exe in common Steam locations")
	}
	if err := s.manager.SetGameDir(dir); err != nil {
		return "", err
	}
	return dir, nil
}

func (s *State) SetGameDir(dir string) error {
	return s.manager.SetGameDir(dir)
}

func (s *State) ClearConfig() error {
	return s.manager.ClearConfig()
}

func (s *State) ListAvailableMods() ([]manager.ModPackage, error) {
	dir := s.manager.GetGameDir()
	if dir == "" {
		detected, err := s.manager.FindGameDir()
		if err != nil {
			return nil, err
		}
		if detected != "" {
			s.manager.Config.GameDir = detected
			_ = s.manager.SaveConfig()
			dir = detected
		}
	}
	if dir == "" {
		dir = filepath.Join(s.manager.BaseDir, "_game_dir_not_configured")
	} else if err := s.manager.SyncLocalMods(dir); err != nil {
		return nil, err
	}
	return s.manager.ListAvailableMods(dir)
}

func (s *State) PreviewZipMods(zipPath string) ([]manager.ArchiveImportCandidate, error) {
	return s.manager.PreviewZipMods(zipPath)
}

func (s *State) ListInstalledMods() ([]manager.InstalledMod, error) {
	dir, err := s.EnsureGameDir()
	if err != nil {
		return nil, err
	}
	return s.manager.ListInstalledMods(dir)
}

func (s *State) InstallMods(mods []manager.ModPackage) ([]manager.InstallResult, error) {
	dir, err := s.EnsureGameDir()
	if err != nil {
		return nil, err
	}
	return s.manager.InstallMods(dir, mods)
}

func (s *State) ImportAvailableModsFromZip(zipPath string) ([]manager.ArchiveImportResult, error) {
	return s.manager.ImportAvailableModsFromZip(zipPath)
}

func (s *State) ImportInstalledModsFromZip(zipPath string) ([]manager.ArchiveImportResult, error) {
	dir, err := s.EnsureGameDir()
	if err != nil {
		return nil, err
	}
	return s.manager.ImportInstalledModsFromZip(dir, zipPath)
}

func (s *State) ExportAvailableModsToZip(mods []manager.ModPackage, zipPath string) (manager.ArchiveExportResult, error) {
	return s.manager.ExportAvailableModsToZip(mods, zipPath)
}

func (s *State) ExportInstalledModsToZip(names []string, zipPath string) (manager.ArchiveExportResult, error) {
	dir, err := s.EnsureGameDir()
	if err != nil {
		return manager.ArchiveExportResult{}, err
	}
	return s.manager.ExportInstalledModsToZip(dir, names, zipPath)
}

func (s *State) UninstallMods(names []string) ([]manager.UninstallResult, bool, error) {
	dir, err := s.EnsureGameDir()
	if err != nil {
		return nil, false, err
	}
	results, err := s.manager.UninstallMods(dir, names)
	if err != nil {
		return nil, false, err
	}
	warn, warnErr := s.manager.ShouldWarnDisableMods(dir, false)
	if warnErr != nil {
		return results, false, warnErr
	}
	return results, warn, nil
}

func (s *State) RepairAvailableMod(sourcePath string) (manager.ModRepairResult, error) {
	return s.manager.RepairMod(sourcePath)
}

func (s *State) RepairInstalledMod(dirName string) (manager.ModRepairResult, error) {
	dir, err := s.EnsureGameDir()
	if err != nil {
		return manager.ModRepairResult{}, err
	}
	return s.manager.RepairMod(filepath.Join(dir, "mods", dirName))
}

func (s *State) UninstallAllMods() (bool, error) {
	dir, err := s.EnsureGameDir()
	if err != nil {
		return false, err
	}
	if err := s.manager.UninstallAllMods(dir); err != nil {
		return false, err
	}
	warn, err := s.manager.ShouldWarnDisableMods(dir, true)
	if err != nil {
		return false, err
	}
	return warn, nil
}

func (s *State) CleanupBakFiles() ([]string, error) {
	dir, err := s.EnsureGameDir()
	if err != nil {
		return nil, err
	}
	return s.manager.CleanupBakFiles(dir)
}

func (s *State) ListSteamIDs() ([]string, error) {
	ids, err := s.manager.ListSteamIDs()
	if err != nil {
		return nil, err
	}
	s.normalizeSelectedSteamID(ids)
	return ids, nil
}

func (s *State) ListSteamProfiles() ([]manager.SteamProfile, error) {
	profiles, err := s.manager.ListSteamProfiles()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		ids = append(ids, profile.SteamID)
	}
	s.normalizeSelectedSteamID(ids)
	return profiles, nil
}

func (s *State) SelectedSteamID() string {
	return s.selectedSteamID
}

func (s *State) SetSelectedSteamID(id string) {
	s.selectedSteamID = id
}

func (s *State) GetSaveSlotInfo(saveType manager.SaveType, slot int) (manager.SaveSlotInfo, error) {
	steamID, err := s.requireSteamID()
	if err != nil {
		return manager.SaveSlotInfo{}, err
	}
	return s.manager.GetSaveSlotInfo(steamID, saveType, slot)
}

func (s *State) ListSaveSlots(saveType manager.SaveType) ([]manager.SaveSlotInfo, error) {
	steamID, err := s.requireSteamID()
	if err != nil {
		return nil, err
	}
	return s.manager.ListSaveSlots(steamID, saveType)
}

func (s *State) CopySave(srcType manager.SaveType, srcSlot int, dstType manager.SaveType, dstSlot int, createBeforeCopyBackup bool) (manager.SaveCopyResult, error) {
	steamID, err := s.requireSteamID()
	if err != nil {
		return manager.SaveCopyResult{}, err
	}
	return s.manager.CopySave(steamID, srcType, srcSlot, dstType, dstSlot, manager.SaveCopyOptions{
		BackupTag:              "auto_before_copy",
		CreateBeforeCopyBackup: createBeforeCopyBackup,
	})
}

func (s *State) BackupSave(saveType manager.SaveType, slot int) (string, error) {
	steamID, err := s.requireSteamID()
	if err != nil {
		return "", err
	}
	return s.manager.BackupSave(steamID, saveType, "manual", slot)
}

func (s *State) ListBackups() ([]manager.BackupInfo, error) {
	steamID, err := s.requireSteamID()
	if err != nil {
		return nil, err
	}
	return s.manager.ListBackups(steamID)
}

func (s *State) RestoreBackup(backup manager.BackupInfo, targetSlot int) (manager.SaveCopyResult, error) {
	steamID, err := s.requireSteamID()
	if err != nil {
		return manager.SaveCopyResult{}, err
	}
	return s.manager.RestoreBackup(steamID, backup, targetSlot)
}

func (s *State) DeleteBackup(backup manager.BackupInfo) error {
	_, err := s.requireSteamID()
	if err != nil {
		return err
	}
	return s.manager.DeleteBackup(backup)
}

func (s *State) requireSteamID() (string, error) {
	if s.selectedSteamID != "" {
		return s.selectedSteamID, nil
	}
	ids, err := s.ListSteamIDs()
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("no Steam save directories found in %s", s.manager.SaveRoot)
	}
	return s.selectedSteamID, nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (s *State) normalizeSelectedSteamID(ids []string) {
	if len(ids) == 0 {
		s.selectedSteamID = ""
		return
	}
	if !containsString(ids, s.selectedSteamID) {
		s.selectedSteamID = ids[0]
	}
}
