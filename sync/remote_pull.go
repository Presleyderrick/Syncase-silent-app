// remote_pull.go
package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"Syncase-silent-app-main/config"
	"Syncase-silent-app-main/crypto"
	"Syncase-silent-app-main/uploader"
)

func InitialSync(ctx context.Context, cfg *config.Config) error {
	fmt.Println("[INITIAL SYNC] Pulling from remote...")

	if err := uploader.SyncRemoteToLocal(ctx, cfg); err != nil {
		return fmt.Errorf("initial sync failed: %w", err)
	}

	// Decrypt all .enc files
	return filepath.Walk(cfg.WatchedFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".enc") {
			out := strings.TrimSuffix(path, ".enc")
			key, keyErr := crypto.LoadKeyFromConfig(cfg.EncryptionKey)
			if keyErr != nil {
				return keyErr
			}
			if err := crypto.DecryptFile(key, path, out); err != nil {
				fmt.Printf("⚠️  Failed to decrypt %s: %v\n", path, err)
			} else {
				os.Remove(path)
			}
		}
		return nil
	})
}
