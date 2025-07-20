package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"syncase/auth"
	"syncase/config"
	"syncase/db"
	"syncase/watcher"
)

func main() {
	if err := runApp(); err != nil {
		fmt.Fprintln(os.Stderr, "❌ App crashed:", err)
		log.Fatal("[FATAL]", err)
	}
}

func runApp() error {
	fmt.Println("🚀 Starting Syncase...")

	// Setup log file
	logFile, err := os.OpenFile("logs/sync.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("❌ Failed to open log file:", err)
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.Println("[INFO] File Sync Service Started")
	fmt.Println("✅ Log file ready at logs/sync.log")

	// Connect to DB
	db.InitDB()

	// Prompt for email + password
	email, password, err := auth.PromptUserCredentials()
	if err != nil {
		fmt.Println("❌ Login cancelled:", err)
		return fmt.Errorf("login cancelled: %w", err)
	}

	user, hash, err := db.GetUserByEmailWithPassword(email)
	if err != nil {
		fmt.Println("❌ Invalid user:", err)
		return fmt.Errorf("user lookup failed: %w", err)
	}

	if !auth.CheckPasswordHash(password, hash) {
		fmt.Println("❌ Incorrect password")
		return fmt.Errorf("incorrect password")
	}

	log.Println("[LOGIN] User logged in:", user.Email)
	fmt.Println("✅ Logged in as:", user.Email)

	// Load config
	fmt.Println("📦 Loading config...")
	cfg := config.Load()

	// Prompt for watched folder if not set
	if cfg.WatchPath == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("🗂 Enter folder path to watch: ")
		pathInput, _ := reader.ReadString('\n')
		pathInput = strings.TrimSpace(pathInput)

		// Check and optionally create folder
		if _, err := os.Stat(pathInput); os.IsNotExist(err) {
			fmt.Println("📁 Folder does not exist. Create it? (y/n):")
			confirm, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(confirm)) == "y" {
				err := os.MkdirAll(pathInput, 0755)
				if err != nil {
					return fmt.Errorf("failed to create folder: %w", err)
				}
				fmt.Println("✅ Folder created.")
			} else {
				return fmt.Errorf("watched folder is required")
			}
		}

		cfg.WatchPath = pathInput
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Println("✅ Watched folder saved to config.")
	}

	// Ensure encrypted output directory exists
	if _, err := os.Stat(cfg.EncryptedPath); os.IsNotExist(err) {
		os.MkdirAll(cfg.EncryptedPath, 0755)
	}

	// Get allowed folders for this user
	folders, err := db.GetAllowedFolders(user.ID)
	if err != nil {
		fmt.Println("❌ Cannot load allowed folders:", err)
		return fmt.Errorf("failed to get folder access: %w", err)
	}
	isAdmin := user.Role == "admin"

	if len(folders) == 0 && !isAdmin {
		fmt.Println("⚠️ No folders assigned for sync.")
		return fmt.Errorf("no folders allowed for user")
	}

	fmt.Println("📁 Allowed folders:", folders)
	log.Printf("[INFO] User is admin: %v, folders: %v\n", isAdmin, folders)

	// Start secure watcher
	fmt.Println("👀 Starting secure folder watcher...")
	if err := watcher.StartWithUserAccess(cfg.WatchPath, folders, isAdmin); err != nil {
		fmt.Println("❌ Watcher failed:", err)
		return fmt.Errorf("watcher failed: %w", err)
	}

	fmt.Println("✅ Syncase running and watching for file changes.")
	return nil
}
