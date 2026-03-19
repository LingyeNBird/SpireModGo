package manager

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func copyRegularFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return err
	}
	if err := ensureDir(filepath.Dir(dstPath)); err != nil {
		return err
	}

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return os.Chtimes(dstPath, time.Now(), info.ModTime())
}

func copyFileWithReplaceFallback(srcPath, dstPath string) (bool, string, error) {
	if err := copyRegularFile(srcPath, dstPath); err == nil {
		return false, "", nil
	}

	if _, statErr := os.Stat(dstPath); statErr != nil {
		return false, "", copyRegularFile(srcPath, dstPath)
	}

	backupName := fmt.Sprintf("%s.bak.%d", filepath.Base(dstPath), rand.New(rand.NewSource(time.Now().UnixNano())).Int())
	backupPath := filepath.Join(filepath.Dir(dstPath), backupName)
	if err := os.Rename(dstPath, backupPath); err != nil {
		return false, "", err
	}
	if err := copyRegularFile(srcPath, dstPath); err != nil {
		return true, backupName, err
	}
	return true, backupName, nil
}

func copyStreamWithReplaceFallback(open func() (io.ReadCloser, error), dstPath string, mode os.FileMode, modTime time.Time) (bool, string, error) {
	if err := copyReaderToFile(open, dstPath, mode, modTime); err == nil {
		return false, "", nil
	}

	if _, statErr := os.Stat(dstPath); statErr != nil {
		return false, "", copyReaderToFile(open, dstPath, mode, modTime)
	}

	backupName := fmt.Sprintf("%s.bak.%d", filepath.Base(dstPath), rand.New(rand.NewSource(time.Now().UnixNano())).Int())
	backupPath := filepath.Join(filepath.Dir(dstPath), backupName)
	if err := os.Rename(dstPath, backupPath); err != nil {
		return false, "", err
	}
	if err := copyReaderToFile(open, dstPath, mode, modTime); err != nil {
		return true, backupName, err
	}
	return true, backupName, nil
}

func copyReaderToFile(open func() (io.ReadCloser, error), dstPath string, mode os.FileMode, modTime time.Time) error {
	reader, err := open()
	if err != nil {
		return err
	}
	defer reader.Close()

	if err := ensureDir(filepath.Dir(dstPath)); err != nil {
		return err
	}
	if mode == 0 {
		mode = 0o644
	}
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, reader); err != nil {
		return err
	}
	if modTime.IsZero() {
		modTime = time.Now()
	}
	return os.Chtimes(dstPath, time.Now(), modTime)
}

func copyDirRecursive(srcDir, dstDir string) (int, error) {
	count := 0
	err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dstDir, rel)
		if d.IsDir() {
			return ensureDir(target)
		}
		if err := copyRegularFile(path, target); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}

func removeDirContents(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}
