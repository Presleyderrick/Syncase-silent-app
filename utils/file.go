package utils

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type QueueItem struct {
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"` // "pending", "failed", "done"
}

const QueueFilePath = "storage/queue.json"

// ===== File Utilities =====

func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func EnsureDir(path string) error {
	if !DirExists(path) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

func FileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// ===== Queue Management =====

func LoadQueue() ([]QueueItem, error) {
	var items []QueueItem

	if !FileExists(QueueFilePath) {
		return []QueueItem{}, nil
	}

	data, err := os.ReadFile(QueueFilePath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &items)
	return items, err
}

func SaveQueue(items []QueueItem) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(QueueFilePath, data, 0644)
}

func AddToQueue(path string) error {
	items, _ := LoadQueue()

	// Avoid duplicate entries
	for _, item := range items {
		if item.Path == path && item.Status == "pending" {
			return nil
		}
	}

	items = append(items, QueueItem{
		Path:      path,
		Timestamp: time.Now().UTC(),
		Status:    "pending",
	})
	return SaveQueue(items)
}

func UpdateQueueStatus(path string, newStatus string) error {
	items, err := LoadQueue()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].Path == path {
			items[i].Status = newStatus
			break
		}
	}
	return SaveQueue(items)
}
