package uploader

import (
	"log"
	"path/filepath"

	"github.com/t3rm1n4l/go-mega"
	"syncase/config"
)

func UploadToMega(filePath string) {
	cfg := config.Load()

	client := mega.New()
	err := client.Login(cfg.MegaEmail, cfg.MegaPassword)
	if err != nil {
		log.Println("[ERROR] MEGA login failed:", err)
		return
	}

	// The UploadFile method handles opening and reading the file.
	// We pass the file path directly.
	// The third argument is for a new name on MEGA; "" uses the original filename.
	_, err = client.UploadFile(filePath, client.FS.GetRoot(), "", nil)
	if err != nil {
		log.Println("[ERROR] Upload to MEGA failed:", err)
		return
	}

	filename := filepath.Base(filePath)
	log.Println("[UPLOAD] File uploaded successfully to MEGA:", filename)
}

// === DOWNLOAD ===
func DownloadFromMega(fileName, downloadToPath string) {
	cfg := config.Load()

	client := mega.New()
	err := client.Login(cfg.MegaEmail, cfg.MegaPassword)
	if err != nil {
		log.Println("[ERROR] MEGA login failed:", err)
		return
	}

	// The Children field is unexported. Use GetChildren() to access them.
	root := client.FS.GetRoot()
	children, err := client.FS.GetChildren(root)
	if err != nil {
		log.Println("[ERROR] Failed to get files from MEGA root:", err)
		return
	}

	var target *mega.Node
	for _, n := range children {
		if n.GetName() == fileName {
			target = n
			break
		}
	}

	if target == nil {
		log.Println("[ERROR] File not found in MEGA:", fileName)
		return
	}

	// The DownloadFile method handles creating and writing to the local file.
	// We pass the destination path directly.
	err = client.DownloadFile(target, downloadToPath, nil)
	if err != nil {
		log.Println("[ERROR] Download from MEGA failed:", err)
		return
	}

	log.Println("[DOWNLOAD] File downloaded successfully:", fileName)
}