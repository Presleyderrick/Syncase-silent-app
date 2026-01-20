// sync/lock.go
package sync

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type FileLock struct {
	lockDir string
}

func NewFileLock() *FileLock {
	lockDir := ".synclocks"
	os.MkdirAll(lockDir, 0755)
	return &FileLock{lockDir: lockDir}
}

func (fl *FileLock) lockFilePath(filePath string) string {
	// Create hash-based lock file name
	hash := sha256.Sum256([]byte(filepath.Clean(filePath)))
	return filepath.Join(fl.lockDir, fmt.Sprintf("%x.lock", hash[:8]))
}

func (fl *FileLock) Acquire(filePath string) (bool, error) {
	lockFile := fl.lockFilePath(filePath)

	file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			info, err := os.Stat(lockFile)
			if err == nil && time.Since(info.ModTime()) > 2*time.Minute {
				os.Remove(lockFile)
				return fl.Acquire(filePath)
			}
			return false, nil
		}
		return false, err
	}
	defer file.Close()

	file.WriteString(time.Now().Format(time.RFC3339))
	return true, nil
}

func (fl *FileLock) Release(filePath string) error {
	return os.Remove(fl.lockFilePath(filePath))
}

func WithLock(filePath string, fn func() error) error {
	lock := NewFileLock()
	acquired, err := lock.Acquire(filePath)
	if err != nil {
		return err
	}
	if !acquired {
		return os.ErrExist
	}
	defer lock.Release(filePath)
	return fn()
}
