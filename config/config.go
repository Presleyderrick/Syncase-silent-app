package config

import (
	"encoding/json"
	"os"
)

type AppConfig struct {
	WatchPath     string `json:"watch_path"`
	EncryptedPath string `json:"encrypted_path"`
	MegaEmail     string `json:"mega_email"`
	MegaPassword  string `json:"mega_password"`
	EncryptionKey string `json:"encryption_key"`
}

const configFile = "syncase_config.json"

func Load() AppConfig {
	var cfg AppConfig

	// Default fallback
	defaultCfg := AppConfig{
		EncryptedPath: "./synced/encrypted_files",
		MegaEmail:     "derrickagida@.com",
		MegaPassword:  "1380Yuare*",
		EncryptionKey: "mySuperSecretKey123",
	}

	file, err := os.ReadFile(configFile)
	if err != nil {
		return defaultCfg // First-time use
	}

	if err := json.Unmarshal(file, &cfg); err != nil {
		return defaultCfg
	}

	// Apply defaults for missing fields
	if cfg.EncryptedPath == "" {
		cfg.EncryptedPath = defaultCfg.EncryptedPath
	}
	if cfg.EncryptionKey == "" {
		cfg.EncryptionKey = defaultCfg.EncryptionKey
	}
	if cfg.MegaEmail == "" {
		cfg.MegaEmail = defaultCfg.MegaEmail
	}
	if cfg.MegaPassword == "" {
		cfg.MegaPassword = defaultCfg.MegaPassword
	}

	return cfg
}

func Save(cfg AppConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}
