package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"Syncase-silent-app-main/config"
	"Syncase-silent-app-main/uploader"
	"Syncase-silent-app-main/watcher"
)

func runMain() error {
	fmt.Println("🚀 Starting Syncase...")

	// Ensure logs directory exists
	if err := os.MkdirAll("logs", 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Setup logging
	logFile, err := os.OpenFile("logs/sync.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.Println("[INFO] Syncase started")

	// Load configuration
	configPath := "config.json"
	fmt.Println("📦 Loading config from:", configPath)
	cfg, err := config.LoadConfigFromFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log.Printf("[INFO] Config loaded: watchedFolder=%s, remote=%s", cfg.WatchedFolder, cfg.RcloneRemote)

	// Resolve absolute path for watched folder
	cfg.WatchedFolder, err = filepath.Abs(cfg.WatchedFolder)
	if err != nil {
		return fmt.Errorf("failed to resolve watched folder path: %w", err)
	}

	// Ensure watched folder exists
	fmt.Println("🗂 Checking watched folder:", cfg.WatchedFolder)
	info, err := os.Stat(cfg.WatchedFolder)
	if err != nil {
		return fmt.Errorf("watched folder error: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("watched path is not a directory: %s", cfg.WatchedFolder)
	}
	fmt.Println("✅ Watched folder ready")

	ctx := context.Background()

	// Initial remote → local sync to match folders
	fmt.Println("🔁 Performing initial remote pull to match folders...")
	if err := uploader.SyncRemoteToLocal(ctx, cfg); err != nil {
		log.Println("[WARN] Initial remote pull failed:", err)
	} else {
		log.Println("[INFO] Remote → Local baseline sync completed")
	}

	// Start watcher (blocking)
	fmt.Println("👀 Starting watcher...")
	log.Println("[INFO] Starting folder watcher")
	if err := watcher.StartWatcher(ctx, cfg); err != nil {
		return fmt.Errorf("watcher exited with error: %w", err)
	}

	return nil
}
