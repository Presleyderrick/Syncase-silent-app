package uploader

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

func UploadToGCS(filePath, bucketName string) {
	ctx := context.Background()


	client, err := storage.NewClient(ctx, option.WithCredentialsFile("syncase-de7486aaeb92.json"))
	if err != nil {
		log.Println("[ERROR] Failed to create GCS client:", err)
		return
	}
	defer client.Close()

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("[ERROR] Failed to open file:", err)
		return
	}
	defer file.Close()

	// Prepare writer to GCS
	objectName := filepath.Base(filePath)
	wc := client.Bucket(bucketName).Object(objectName).NewWriter(ctx)

	if _, err = wc.Write([]byte{}); err != nil {
		log.Println("[ERROR] Failed to start upload:", err)
		return
	}
	if _, err := file.Seek(0, 0); err != nil {
		log.Println("[ERROR] Failed to rewind file:", err)
		return
	}
	if _, err = file.WriteTo(wc); err != nil {
		log.Println("[ERROR] Failed to upload:", err)
		return
	}

	if err := wc.Close(); err != nil {
		log.Println("[ERROR] Failed to finalize upload:", err)
		return
	}

	log.Println("[UPLOAD] File uploaded to Google Cloud Storage:", objectName)
}
