package manager

import "time"

const (
	sts2AppID   = "2868840"
	sts2DirName = "Slay the Spire 2"
	sts2ExeName = "SlayTheSpire2.exe"
)

type SaveType string

const (
	SaveTypeNormal SaveType = "normal"
	SaveTypeModded SaveType = "modded"
)

type Config struct {
	GameDir string `json:"GameDir"`
}

type ModManifest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Author  string `json:"author"`
	PckName string `json:"pck_name"`
}

type ModPackage struct {
	DirName          string
	SourcePath       string
	InstallName      string
	Label            string
	Manifest         *ModManifest
	Installed        bool
	InstalledVersion string
	Updatable        bool
}

type InstalledMod struct {
	DirName  string
	FullPath string
	Manifest *ModManifest
	Label    string
}

type InstallFileResult struct {
	Name       string
	Replaced   bool
	BackupName string
	Err        error
}

type InstallResult struct {
	Mod           ModPackage
	Files         []InstallFileResult
	FilesCopied   int
	EnableChanged bool
}

type SaveSlotInfo struct {
	Type          SaveType
	Slot          int
	Path          string
	HasData       bool
	LastModified  time.Time
	HasCurrentRun bool
}

type SaveCopyOptions struct {
	BackupTag              string
	CreateBeforeCopyBackup bool
}

type SaveCopyResult struct {
	CopiedFiles  int
	BackupDir    string
	CloudSynced  bool
	CloudUpdated int
}

type BackupInfo struct {
	Name      string
	FullPath  string
	Type      SaveType
	Slot      int
	FileCount int
}
