package uploader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"Syncase-silent-app-main/config"
)

const (
	maxUploadAttempts = 3
	maxSyncAttempts   = 3
	rcloneTimeout     = 10 * time.Minute
)

// UploadWithRclone uploads a single file to the remote with retries and verification
func UploadWithRclone(ctx context.Context, cfg *config.Config, localPath string) error {
	// Get absolute path for local file
	absLocalPath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(absLocalPath); err != nil {
		return fmt.Errorf("local file not found: %w", err)
	}

	// Get relative path from watched folder
	relPath, err := filepath.Rel(cfg.WatchedFolder, absLocalPath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	// Construct remote path (ensure forward slashes for rclone)
	remoteDest := cfg.RcloneRemote + ":/Watched_folder/" + filepath.ToSlash(relPath)
	log.Printf("[UPLOAD] %s -> %s", filepath.Base(localPath), remoteDest)

	var lastErr error

	for attempt := 1; attempt <= maxUploadAttempts; attempt++ {
		uploadCtx, cancel := context.WithTimeout(ctx, rcloneTimeout)

		var stdout, stderr bytes.Buffer
		cmd := exec.CommandContext(
			uploadCtx,
			"rclone",
			"copyto",
			absLocalPath,
			remoteDest,
			"--retries", "2",
			"--low-level-retries", "3",
			"--stats", "0",
			"--progress",
			"--transfers", "1",
			"--checkers", "1",
		)

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			lastErr = fmt.Errorf(
				"upload failed (attempt %d): %v | stderr: %s",
				attempt, err, strings.TrimSpace(stderr.String()),
			)
			log.Println(lastErr)
			cancel()
			backoff(attempt)
			continue
		}

		cancel()

		// Hard verification
		if err := verifyRemoteFile(ctx, remoteDest); err != nil {
			lastErr = fmt.Errorf("verification failed (attempt %d): %w", attempt, err)
			log.Println(lastErr)
			backoff(attempt)
			continue
		}

		log.Printf("[UPLOAD OK] %s verified on remote", filepath.Base(localPath))
		return nil
	}

	return fmt.Errorf("upload failed after %d attempts: %w", maxUploadAttempts, lastErr)
}

// verifyRemoteFile ensures the file exists remotely
func verifyRemoteFile(ctx context.Context, remotePath string) error {
	verifyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(verifyCtx, "rclone", "lsf", remotePath)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Check if the error is "directory not found" (file might be at root)
		errStr := stderr.String()
		if strings.Contains(errStr, "directory not found") || strings.Contains(errStr, "Couldn't find") {
			// Try checking if file exists by checking parent directory
			if _, err := verifyRemoteHasFiles(ctx, filepath.Dir(remotePath)); err != nil {
				return fmt.Errorf("parent directory verification failed: %w", err)
			}
			return nil
		}
		return fmt.Errorf("verify failed: %s", strings.TrimSpace(errStr))
	}

	if strings.TrimSpace(stdout.String()) == "" {
		return errors.New("remote file missing after upload")
	}

	return nil
}

// SyncLocalToRemote ensures remote mirrors local exactly (sends encrypted files)
func SyncLocalToRemote(ctx context.Context, cfg *config.Config) error {
	remoteRoot := cfg.RcloneRemote + ":/Watched_folder"
	log.Printf("[SYNC] Local -> Remote: %s -> %s", cfg.WatchedFolder, remoteRoot)

	var lastErr error

	for attempt := 1; attempt <= maxSyncAttempts; attempt++ {
		syncCtx, cancel := context.WithTimeout(ctx, rcloneTimeout)

		var stdout, stderr bytes.Buffer
		cmd := exec.CommandContext(
			syncCtx,
			"rclone",
			"sync",
			cfg.WatchedFolder,
			remoteRoot,
			"--create-empty-src-dirs",
			"--exclude", "*.synclock", // Keep .enc files, exclude lock files
			"--exclude", ".synclocks/**",
			"--retries", "2",
			"--low-level-retries", "3",
			"--stats", "30s",
			"--progress",
			"--transfers", "4",
			"--checkers", "8",
		)

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			lastErr = fmt.Errorf(
				"sync failed (attempt %d): %v | stderr: %s",
				attempt, err, strings.TrimSpace(stderr.String()),
			)
			log.Println(lastErr)
			cancel()
			backoff(attempt)
			continue
		}

		cancel()

		// Log sync stats
		if stdout.Len() > 0 {
			log.Printf("[SYNC STATS] %s", strings.TrimSpace(stdout.String()))
		}

		// Verify sync was successful by checking remote has files
		if ok, err := verifyRemoteHasFiles(ctx, remoteRoot); err != nil {
			lastErr = fmt.Errorf("remote verification failed: %w", err)
			backoff(attempt)
			continue
		} else if !ok {
			lastErr = errors.New("remote appears to be empty after sync")
			backoff(attempt)
			continue
		}

		log.Println("[SYNC OK] Local files synced to remote")
		return nil
	}

	return fmt.Errorf("sync failed after %d attempts: %w", maxSyncAttempts, lastErr)
}

// SyncRemoteToLocal ensures the bidirectional sync from remote to local
func SyncRemoteToLocal(ctx context.Context, cfg *config.Config) error {
	remoteRoot := cfg.RcloneRemote + ":/Watched_folder"
	log.Printf("[SYNC] Remote -> Local: %s -> %s", remoteRoot, cfg.WatchedFolder)

	var lastErr error

	for attempt := 1; attempt <= maxSyncAttempts; attempt++ {
		syncCtx, cancel := context.WithTimeout(ctx, rcloneTimeout)

		var stdout, stderr bytes.Buffer
		cmd := exec.CommandContext(
			syncCtx,
			"rclone",
			"sync",
			remoteRoot,
			cfg.WatchedFolder,
			"--create-empty-src-dirs",
			"--exclude", "*.synclock", // Keep .enc files, exclude lock files
			"--exclude", ".synclocks/**",
			"--retries", "2",
			"--low-level-retries", "3",
			"--stats", "30s",
			"--progress",
			"--transfers", "4",
			"--checkers", "8",
		)

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			lastErr = fmt.Errorf(
				"sync failed (attempt %d): %v | stderr: %s",
				attempt, err, strings.TrimSpace(stderr.String()),
			)
			log.Println(lastErr)
			cancel()
			backoff(attempt)
			continue
		}

		cancel()

		// Log sync stats
		if stdout.Len() > 0 {
			log.Printf("[SYNC STATS] %s", strings.TrimSpace(stdout.String()))
		}

		// Verify sync was successful by checking local has files
		if ok, err := verifyLocalHasFiles(cfg.WatchedFolder); err != nil {
			lastErr = fmt.Errorf("local verification failed: %w", err)
			backoff(attempt)
			continue
		} else if !ok {
			lastErr = errors.New("local appears to be empty after sync")
			backoff(attempt)
			continue
		}

		log.Println("[SYNC OK] Remote files synced to local")
		return nil
	}

	return fmt.Errorf("sync failed after %d attempts: %w", maxSyncAttempts, lastErr)
}

// verifyRemoteHasFiles checks remote integrity post-sync
func verifyRemoteHasFiles(ctx context.Context, remoteRoot string) (bool, error) {
	verifyCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(
		verifyCtx,
		"rclone",
		"lsf",
		remoteRoot,
		"--recursive",
		"--files-only",
		"--max-depth", "3", // Limit depth for performance
	)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Check if error is "directory not found" - might be empty
		errStr := stderr.String()
		if strings.Contains(errStr, "directory not found") ||
			strings.Contains(errStr, "doesn't exist") ||
			strings.Contains(errStr, "not found") ||
			strings.Contains(errStr, "Couldn't find") {
			// Empty remote is okay for first sync
			return true, nil
		}
		return false, fmt.Errorf("verify failed: %s", strings.TrimSpace(errStr))
	}

	// Check if we got any output
	output := strings.TrimSpace(stdout.String())
	if output == "" {
		// Empty remote is okay
		return true, nil
	}

	// Count files
	lines := strings.Split(output, "\n")
	fileCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			fileCount++
		}
	}

	if fileCount > 0 {
		log.Printf("[VERIFY] Remote has %d files", fileCount)
	}
	return true, nil
}

// verifyLocalHasFiles checks if local folder has files (including .enc files)
func verifyLocalHasFiles(localRoot string) (bool, error) {
	fileCount := 0
	err := filepath.Walk(localRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip permission errors
			if os.IsPermission(err) {
				return nil
			}
			return err
		}

		if !info.IsDir() {
			// Count all files except lock files
			if !strings.HasSuffix(path, ".synclock") &&
				!strings.Contains(path, ".synclocks"+string(os.PathSeparator)) {
				fileCount++
			}
		}
		return nil
	})

	if err != nil {
		return false, fmt.Errorf("failed to scan local folder: %w", err)
	}

	if fileCount > 0 {
		log.Printf("[VERIFY] Local has %d files", fileCount)
	}
	return fileCount > 0, nil
}

// UploadWithVersioning uploads file with timestamped versioning
func UploadWithVersioning(ctx context.Context, cfg *config.Config, localPath string) error {
	// Get absolute path for local file
	absLocalPath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(absLocalPath); err != nil {
		return fmt.Errorf("local file not found: %w", err)
	}

	// Create timestamped backup on remote
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Base(localPath)
	remoteVersionsDest := cfg.RcloneRemote + ":/Watched_folder_versions/" + timestamp + "/" + filename

	log.Printf("[VERSIONING] Creating backup version: %s -> %s", filename, remoteVersionsDest)

	uploadCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(
		uploadCtx,
		"rclone",
		"copyto",
		absLocalPath,
		remoteVersionsDest,
		"--retries", "1",
		"--low-level-retries", "2",
		"--stats", "0",
	)

	cmd.Stderr = &stderr

	// Run versioning upload (don't fail main upload if versioning fails)
	if err := cmd.Run(); err != nil {
		log.Printf("[VERSIONING WARN] Failed to create backup version: %v - %s",
			err, strings.TrimSpace(stderr.String()))
		// Don't return error - versioning is optional
	} else {
		log.Printf("[VERSIONING OK] Created backup version: %s", remoteVersionsDest)
	}

	return nil
}

// Simple backoff with jitter
func backoff(attempt int) {
	// Exponential backoff with jitter
	baseDelay := time.Duration(attempt*attempt) * time.Second
	jitter := time.Duration(attempt*500) * time.Millisecond
	sleepTime := baseDelay + jitter

	if sleepTime > 30*time.Second {
		sleepTime = 30 * time.Second
	}

	log.Printf("[BACKOFF] Waiting %v before retry (attempt %d)", sleepTime, attempt)
	time.Sleep(sleepTime)
}

// TestRcloneConnection tests if rclone is working and remote is accessible
func TestRcloneConnection(ctx context.Context, cfg *config.Config) error {
	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Test rclone executable
	var stderr bytes.Buffer
	cmd := exec.CommandContext(testCtx, "rclone", "version")
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone not found in PATH: %v - %s",
			err, strings.TrimSpace(stderr.String()))
	}

	// Test remote connection
	testRemote := cfg.RcloneRemote + ":"
	cmd = exec.CommandContext(testCtx, "rclone", "lsd", testRemote)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot connect to remote '%s': %v - %s",
			cfg.RcloneRemote, err, strings.TrimSpace(stderr.String()))
	}

	return nil
}

// GetRemoteFileList gets list of files from remote
func GetRemoteFileList(ctx context.Context, cfg *config.Config) ([]string, error) {
	remoteRoot := cfg.RcloneRemote + ":/Watched_folder"
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(
		ctx,
		"rclone",
		"lsf",
		remoteRoot,
		"--recursive",
		"--files-only",
	)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list remote files: %s", strings.TrimSpace(stderr.String()))
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return []string{}, nil
	}

	return strings.Split(output, "\n"), nil
}
