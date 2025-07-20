package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"syncase/crypto"
	"syncase/downloader"
	"syncase/uploader"
)

// Accepts user-specific access rules with admin override
func StartWithUserAccess(watchPath string, allowedFolders []string, isAdmin bool) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = watcher.Add(watchPath)
	if err != nil {
		return err
	}

	log.Println("[INFO] Watching folder:", watchPath)

	// Start background poller to fetch server files
	go startSyncPoller(watchPath, allowedFolders, isAdmin)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				log.Println("[EVENT] File change detected:", event.Name)

				// Get relative path
				relPath := strings.TrimPrefix(event.Name, watchPath+string(os.PathSeparator))

				// Check if user has access
				if !isAdmin && !isAllowed(relPath, allowedFolders) {
					log.Println("[BLOCKED] Access denied to file:", relPath)
					continue
				}

				// Wait until file is accessible
				if !waitUntilAccessible(event.Name, 10, 500*time.Millisecond) {
					log.Println("[ERROR] File still locked, skipping:", event.Name)
					continue
				}

				// Encrypt and upload
				encryptedPath := filepath.Join("./synced/encrypted_files", filepath.Base(event.Name)+".enc")
				err := crypto.EncryptFile(event.Name, encryptedPath)
				if err != nil {
					log.Println("[ERROR] Encryption failed:", err)
					continue
				}

				go uploader.UploadToB2(encryptedPath)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Println("[ERROR] Watcher error:", err)
		}
	}
}

// Check if relative file path starts with one of the allowed folder prefixes
func isAllowed(path string, allowedFolders []string) bool {
	for _, prefix := range allowedFolders {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// Retry logic for locked files
func waitUntilAccessible(path string, retries int, delay time.Duration) bool {
	for i := 0; i < retries; i++ {
		f, err := os.Open(path)
		if err == nil {
			f.Close()
			return true
		}

		if strings.Contains(err.Error(), "being used by another process") || os.IsNotExist(err) {
			time.Sleep(delay)
			continue
		}

		log.Printf("[ERROR] Cannot access file: %v\n", err)
		return false
	}
	return false
}

// Poll server periodically for new files and download to watchPath
func startSyncPoller(watchPath string, allowedFolders []string, isAdmin bool) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("[SYNC] Checking for server-side updates...")
		err := syncfetch.FetchAndApplyServerFiles(watchPath, allowedFolders, isAdmin)
		if err != nil {
			log.Println("[ERROR] Sync poller failed:", err)
		}
	}
}
