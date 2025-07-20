package syncfetch

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"syncase/crypto"
)

var (
	B2Region    = "eu-central-003"
	B2Endpoint  = "https://s3.eu-central-003.backblazeb2.com"
	BucketName  = "SyncaseKBP"
	B2AccessKey = "0035e4a3e38a1e00000000002"
	B2SecretKey = "K003SItSZjLC0ht9DxepQzQM3XLwqmU"
)

func FetchAndApplyServerFiles(localPath string, allowedFolders []string, isAdmin bool) error {
	cfg := aws.Config{
		Region:      B2Region,
		Credentials: credentials.NewStaticCredentialsProvider(B2AccessKey, B2SecretKey, ""),
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:           B2Endpoint,
				SigningRegion: B2Region,
			}, nil
		}),
	}
	client := s3.NewFromConfig(cfg)

	// List files in bucket
	resp, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(BucketName),
	})
	if err != nil {
		return err
	}

	for _, obj := range resp.Contents {
		key := *obj.Key
		if !strings.HasSuffix(key, ".enc") {
			continue
		}

		// Check access
		base := strings.TrimSuffix(filepath.Base(key), ".enc")
		if !isAdmin && !isInAllowedFolders(base, allowedFolders) {
			continue
		}

		// Download
		outPath := filepath.Join(localPath, base)
		tmpEnc := outPath + ".enc"

		outFile, err := os.Create(tmpEnc)
		if err != nil {
			log.Println("[ERROR] Creating temp file failed:", err)
			continue
		}

		getObj, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: aws.String(BucketName),
			Key:    aws.String(key),
		})
		if err != nil {
			log.Println("[ERROR] Download failed:", err)
			outFile.Close()
			continue
		}

		_, err = io.Copy(outFile, getObj.Body)
		outFile.Close()
		if err != nil {
			log.Println("[ERROR] Failed to save file:", err)
			continue
		}

		// Decrypt
		err = crypto.DecryptFile(tmpEnc, outPath)
		if err != nil {
			log.Println("[ERROR] Decryption failed for", tmpEnc, ":", err)
			continue
		}

		log.Println("[SYNC] Synced file:", outPath)
	}

	return nil
}

func isInAllowedFolders(name string, allowed []string) bool {
	for _, folder := range allowed {
		if strings.HasPrefix(name, folder) {
			return true
		}
	}
	return false
}
