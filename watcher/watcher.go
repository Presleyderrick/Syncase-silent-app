package watcher

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	syncstd "sync"
	"time"

	"Syncase-silent-app-main/config"
	"Syncase-silent-app-main/crypto"
	syncpkg "Syncase-silent-app-main/sync"
	"Syncase-silent-app-main/uploader"

	"github.com/fsnotify/fsnotify"
)

const (
	fileStableInterval = 2 * time.Second
	fileStableTries    = 3
	syncDebounceTime   = 10 * time.Second
	maxLockWaitTime    = 30 * time.Second
	maxWatchDepth      = 6     // Maximum depth to watch
	watchBufferSize    = 10000 // Large buffer for many directories
	watchWorkers       = 4     // Parallel workers for adding watches
)

// StartWatcher starts watching the local folder and syncing changes to remote
func StartWatcher(ctx context.Context, cfg *config.Config) error {
	log.Println("[WATCHER] Starting optimized watcher...")

	// Load encryption key
	key, err := crypto.LoadKeyFromConfig(cfg.EncryptionKey)
	if err != nil {
		return err
	}

	// Initialize fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// Create file lock instance
	fileLock := syncpkg.NewFileLock()

	var (
		mu          syncstd.Mutex
		syncRunning bool
		pendingOps  = make(map[string]time.Time)
	)

	// Create priority channels for watch additions
	highPriorityCh := make(chan string, 1000)
	medPriorityCh := make(chan string, 3000)
	lowPriorityCh := make(chan string, 6000)

	// Start watch workers
	for i := 0; i < watchWorkers; i++ {
		go watchWorker(ctx, watcher, highPriorityCh, medPriorityCh, lowPriorityCh, i)
	}

	// Initial watch setup with prioritization
	go initialWatchSetup(ctx, cfg.WatchedFolder, highPriorityCh, medPriorityCh, lowPriorityCh)

	log.Println("[WATCHER] Watching folder:", cfg.WatchedFolder)

	// Clean up pending ops periodically
	go cleanupPendingOps(ctx, &mu, pendingOps)

	triggerSync := func() {
		mu.Lock()
		if syncRunning {
			mu.Unlock()
			return
		}
		syncRunning = true
		mu.Unlock()

		go func() {
			defer func() {
				mu.Lock()
				syncRunning = false
				mu.Unlock()
			}()

			time.Sleep(syncDebounceTime)

			log.Println("[SYNC] Debounced sync starting...")
			if err := uploader.SyncLocalToRemote(ctx, cfg); err != nil {
				log.Println("[SYNC ERROR]", err)
			} else {
				log.Println("[SYNC] Completed successfully")
			}
		}()
	}

	processFileEvent := func(path string, isDelete bool) {
		// Skip temporary encrypted files
		if strings.HasSuffix(path, ".enc") {
			return
		}

		// Skip if we're ignoring local events (during remote pull)
		if cfg.IgnoreLocalEvents {
			return
		}

		// Check if this is a directory
		info, err := os.Stat(path)
		if err != nil && !isDelete {
			// File may have been deleted already
			return
		}

		// Handle directories - add to watch with priority
		if err == nil && info.IsDir() {
			if !isDelete {
				addDirectoryToWatch(path, cfg.WatchedFolder, highPriorityCh, medPriorityCh, lowPriorityCh)
			}
			triggerSync()
			return
		}

		// Check if this operation is already pending
		mu.Lock()
		if lastOp, exists := pendingOps[path]; exists && time.Since(lastOp) < 5*time.Second {
			mu.Unlock()
			log.Printf("[DUPLICATE] Skipping duplicate event for: %s", path)
			return
		}
		pendingOps[path] = time.Now()
		mu.Unlock()

		// Clean up pending op when done
		defer func() {
			mu.Lock()
			delete(pendingOps, path)
			mu.Unlock()
		}()

		// Handle file operations
		if isDelete {
			log.Printf("[DELETE] %s", path)
			triggerSync()
			return
		}

		// Process file with locking
		go processFileWithLock(ctx, path, key, cfg, fileLock, &mu, triggerSync)
	}

	for {
		select {
		case <-ctx.Done():
			close(highPriorityCh)
			close(medPriorityCh)
			close(lowPriorityCh)
			log.Println("[WATCHER] Shutting down...")
			return nil

		case ev := <-watcher.Events:
			// Skip if we're ignoring local events
			if cfg.IgnoreLocalEvents {
				continue
			}

			// Log the event for debugging
			log.Printf("[EVENT] %s: %v", ev.Name, ev.Op)

			// Handle file deletions and renames
			if ev.Op&fsnotify.Remove == fsnotify.Remove || ev.Op&fsnotify.Rename == fsnotify.Rename {
				processFileEvent(ev.Name, true)
				continue
			}

			// Handle file creates/writes/chmods
			if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Chmod) != 0 {
				processFileEvent(ev.Name, false)
			}

		case err := <-watcher.Errors:
			log.Println("[WATCHER ERROR]", err)
		}
	}
}

// ========== Helper Functions ==========

func watchWorker(ctx context.Context, watcher *fsnotify.Watcher, highCh, medCh, lowCh <-chan string, workerID int) {
	// Process high priority first, then medium, then low
	for {
		select {
		case path := <-highCh:
			addWatchWithRetry(watcher, path, workerID, "HIGH")
		case path := <-medCh:
			// Small delay for medium priority
			time.Sleep(10 * time.Millisecond)
			addWatchWithRetry(watcher, path, workerID, "MED")
		case path := <-lowCh:
			// Longer delay for low priority
			time.Sleep(50 * time.Millisecond)
			addWatchWithRetry(watcher, path, workerID, "LOW")
		case <-ctx.Done():
			return
		}
	}
}

func addWatchWithRetry(watcher *fsnotify.Watcher, path string, workerID int, priority string) {
	for i := 0; i < 3; i++ {
		if err := watcher.Add(path); err != nil {
			log.Printf("[WORKER %d %s PRIO ERROR] %s: %v", workerID, priority, path, err)
			time.Sleep(time.Duration(i*100) * time.Millisecond)
			continue
		}
		// Only log successful additions for high priority to reduce noise
		if priority == "HIGH" {
			log.Printf("[WORKER %d %s PRIO] Added: %s", workerID, priority, path)
		}
		return
	}
}

func initialWatchSetup(ctx context.Context, root string, highCh, medCh, lowCh chan<- string) {
	log.Println("[INITIAL WATCH] Starting prioritized directory scan...")

	// First, add the root folder to high priority
	select {
	case highCh <- root:
		log.Printf("[INITIAL WATCH] Root added: %s", root)
	default:
		log.Printf("[INITIAL WATCH] Could not add root to queue: %s", root)
	}

	// Walk and prioritize subdirectories
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}

		// Skip hidden directories
		if d.Name()[0] == '.' && d.Name() != "." && d.Name() != ".." {
			return filepath.SkipDir
		}

		// Skip root (already added)
		if path == root {
			return nil
		}

		// Determine priority based on depth and name
		priority := determinePriority(path, root)

		// Send to appropriate channel
		switch priority {
		case 1:
			select {
			case highCh <- path:
			default:
				log.Printf("[INITIAL WATCH HIGH PRIO QUEUE FULL] %s", path)
			}
		case 2:
			select {
			case medCh <- path:
			default:
				log.Printf("[INITIAL WATCH MED PRIO QUEUE FULL] %s", path)
			}
		case 3:
			select {
			case lowCh <- path:
			default:
				log.Printf("[INITIAL WATCH LOW PRIO QUEUE FULL] %s", path)
			}
		default:
			// Priority 0 or negative - don't watch
			if priority == 0 {
				return nil
			}
			// If depth > maxWatchDepth, skip the entire subtree
			return filepath.SkipDir
		}

		return nil
	})

	log.Println("[INITIAL WATCH] Scan complete")
}

func determinePriority(path, root string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return 0
	}

	depth := strings.Count(rel, string(os.PathSeparator))
	dirName := strings.ToLower(filepath.Base(path))

	// Check if too deep
	if depth > maxWatchDepth {
		return -1 // Negative means skip entire subtree
	}

	// High priority: shallow directories and important ones
	if depth <= 2 {
		return 1
	}

	// Medium priority: moderately deep but important
	if depth <= 4 {
		// Check for important patterns
		importantKeywords := []string{"active", "current", "202", "client", "matter", "case", "urgent"}
		for _, keyword := range importantKeywords {
			if strings.Contains(dirName, keyword) {
				return 2
			}
		}
	}

	// Low priority: everything else within depth limit
	if depth <= maxWatchDepth {
		return 3
	}

	return 0 // Don't watch
}

func addDirectoryToWatch(path, root string, highCh, medCh, lowCh chan<- string) {
	priority := determinePriority(path, root)

	switch priority {
	case 1:
		select {
		case highCh <- path:
		default:
			log.Printf("[HIGH PRIO QUEUE FULL NEW DIR] %s", path)
		}
	case 2:
		select {
		case medCh <- path:
		default:
			log.Printf("[MED PRIO QUEUE FULL NEW DIR] %s", path)
		}
	case 3:
		select {
		case lowCh <- path:
		default:
			log.Printf("[LOW PRIO QUEUE FULL NEW DIR] %s", path)
		}
	case -1:
		// Too deep, don't watch
		return
	default:
		// Priority 0, don't watch
		return
	}
}

func cleanupPendingOps(ctx context.Context, mu *syncstd.Mutex, pendingOps map[string]time.Time) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mu.Lock()
			now := time.Now()
			for path, timestamp := range pendingOps {
				if now.Sub(timestamp) > 10*time.Minute {
					delete(pendingOps, path)
					log.Println("[CLEANUP] Removed stale pending op:", path)
				}
			}
			mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

func processFileWithLock(ctx context.Context, filePath string, key []byte, cfg *config.Config,
	fileLock *syncpkg.FileLock, mu *syncstd.Mutex, triggerSync func()) {

	// Try to acquire lock with timeout
	lockAcquired := false
	lockStart := time.Now()

	for time.Since(lockStart) < maxLockWaitTime {
		acquired, err := fileLock.Acquire(filePath)
		if err != nil {
			log.Printf("[LOCK ERROR] %s: %v", filePath, err)
			return
		}
		if acquired {
			lockAcquired = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !lockAcquired {
		log.Printf("[LOCK TIMEOUT] Could not acquire lock for: %s", filePath)
		return
	}

	defer func() {
		if err := fileLock.Release(filePath); err != nil {
			log.Printf("[LOCK RELEASE ERROR] %s: %v", filePath, err)
		}
	}()

	// Wait for file to be stable
	if err := waitForStable(filePath); err != nil {
		log.Println("[STABLE ERROR]", err)
		return
	}

	// Create encrypted version
	encPath := filePath + ".enc"
	if err := crypto.EncryptFile(key, filePath, encPath); err != nil {
		log.Println("[ENCRYPT ERROR]", err)
		return
	}
	defer func() {
		if err := os.Remove(encPath); err != nil && !os.IsNotExist(err) {
			log.Println("[CLEANUP ERROR] failed to remove temp file:", encPath, err)
		}
	}()

	// Upload encrypted file
	if err := uploader.UploadWithRclone(ctx, cfg, encPath); err != nil {
		log.Println("[UPLOAD ERROR]", err)
		return
	}
	log.Println("[UPLOAD] Uploaded and verified:", filepath.Base(filePath))

	triggerSync()
}

// waitForStable waits until the file size is unchanged for N intervals
func waitForStable(path string) error {
	var lastSize int64 = -1
	stableCount := 0

	for i := 0; i < fileStableTries*3; i++ {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Size() == lastSize {
			stableCount++
			if stableCount >= 2 {
				return nil
			}
		} else {
			stableCount = 0
			lastSize = info.Size()
		}
		time.Sleep(fileStableInterval)
	}

	return nil
}
