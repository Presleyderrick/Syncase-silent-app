package config

import (
	"encoding/json"
	"os"
)

type ConflictStrategy string

const (
	ConflictLocalWins  ConflictStrategy = "local"
	ConflictRemoteWins ConflictStrategy = "remote"
	ConflictNewerWins  ConflictStrategy = "newer"
	ConflictManual     ConflictStrategy = "manual"
)

type Config struct {
	WatchedFolder     string `json:"watchedFolder"`
	RcloneRemote      string `json:"rclone_remote"`
	EncryptionKey     string `json:"encryption_key"`
	MaxDepth          int    `json:"max_depth"`
	IgnoreLocalEvents bool   `json:"-"`
}

func LoadConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
