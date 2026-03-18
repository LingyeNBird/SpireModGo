package manager

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func (m *Manager) SyncSteamCloudCache(steamID string, saveType SaveType, slot int, srcPath string) (bool, int, error) {
	cachePath, err := m.steamCloudCachePath(steamID)
	if err != nil || cachePath == "" {
		return false, 0, err
	}
	remotePath := filepath.Join(cachePath, "remote")
	if !dirExists(remotePath) {
		return false, 0, nil
	}
	cloudRelBase := fmt.Sprintf("profile%d\\saves", slot)
	if saveType == SaveTypeModded {
		cloudRelBase = fmt.Sprintf("modded\\profile%d\\saves", slot)
	}
	cloudDir := filepath.Join(remotePath, cloudRelBase)
	if err := ensureDir(cloudDir); err != nil {
		return false, 0, err
	}

	updated := 0
	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return false, 0, err
	}
	for _, entry := range entries {
		srcItem := filepath.Join(srcPath, entry.Name())
		dstItem := filepath.Join(cloudDir, entry.Name())
		if entry.IsDir() {
			count, err := copyDirRecursive(srcItem, dstItem)
			if err != nil {
				return false, updated, err
			}
			updated += count
			continue
		}
		if err := copyRegularFile(srcItem, dstItem); err != nil {
			return false, updated, err
		}
		updated++
	}

	vdfPath := filepath.Join(cachePath, "remotecache.vdf")
	if fileExists(vdfPath) {
		if err := updateRemoteCacheVDF(vdfPath, cloudRelBase, cloudDir); err != nil {
			m.logf("update remotecache.vdf failed: %v", err)
		}
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		cloudFile := filepath.Join(cloudDir, entry.Name())
		localFile := filepath.Join(srcPath, entry.Name())
		cloudInfo, err := os.Stat(cloudFile)
		if err != nil {
			continue
		}
		_ = os.Chtimes(localFile, time.Now(), cloudInfo.ModTime())
	}
	return true, updated, nil
}

func (m *Manager) steamCloudCachePath(steamID string) (string, error) {
	steamPath := m.getSteamPath()
	if steamPath == "" {
		return "", nil
	}
	id64, err := strconv.ParseInt(steamID, 10, 64)
	if err != nil {
		return "", err
	}
	id32 := id64 - 76561197960265728
	cachePath := filepath.Join(steamPath, "userdata", strconv.FormatInt(id32, 10), sts2AppID)
	if !dirExists(cachePath) {
		return "", nil
	}
	return cachePath, nil
}

func updateRemoteCacheVDF(vdfPath, relBase, cloudDir string) error {
	contentBytes, err := os.ReadFile(vdfPath)
	if err != nil {
		return err
	}
	content := string(contentBytes)
	files := make([]string, 0)
	err = filepath.WalkDir(cloudDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	nowUnix := strconv.FormatInt(time.Now().Unix(), 10)
	normalizedBase := strings.ReplaceAll(relBase, `\`, "/")
	for _, filePath := range files {
		relFile, err := filepath.Rel(cloudDir, filePath)
		if err != nil {
			return err
		}
		vdfKey := normalizedBase + "/" + strings.ReplaceAll(relFile, `\`, "/")
		blockRE := regexp.MustCompile(`(?s)("` + regexp.QuoteMeta(vdfKey) + `"\s*\{)(.*?)(\})`)
		match := blockRE.FindStringSubmatchIndex(content)
		if match == nil {
			continue
		}
		block := content[match[4]:match[5]]
		sha, err := fileSHA1(filePath)
		if err != nil {
			return err
		}
		info, err := os.Stat(filePath)
		if err != nil {
			return err
		}
		block = replaceVDFField(block, "size", strconv.FormatInt(info.Size(), 10))
		block = replaceVDFField(block, "localtime", nowUnix)
		block = replaceVDFField(block, "time", nowUnix)
		block = replaceVDFField(block, "sha", sha)
		block = replaceVDFField(block, "syncstate", "4")
		content = content[:match[4]] + block + content[match[5]:]
	}
	return os.WriteFile(vdfPath, []byte(content), 0o644)
}

func replaceVDFField(block, field, value string) string {
	fieldRE := regexp.MustCompile(`("` + regexp.QuoteMeta(field) + `"\s+")[^"]*(")`)
	return fieldRE.ReplaceAllString(block, `${1}`+value+`${2}`)
}

func fileSHA1(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha1.Sum(data)
	return hex.EncodeToString(sum[:]), nil
}
