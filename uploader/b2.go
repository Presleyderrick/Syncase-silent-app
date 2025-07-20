package uploader

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var (
	B2Region     = "eu-central-003"
	B2Endpoint   = "https://s3.eu-central-003.backblazeb2.com"
	BucketName   = "SyncaseKBP"
	B2AccessKey  = "0035e4a3e38a1e00000000002"
	B2SecretKey  = "K003SItSZjLC0ht9DxepQzQM3XLwqmU"
	B2KeyName    = "KennedyBalala"
)

func UploadToB2(filePath string) {
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

	file, err := os.Open(filePath)
	if err != nil {
		log.Println("[ERROR] Failed to open file:", err)
		return
	}
	defer file.Close()

	objectKey := filepath.Base(filePath)

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(BucketName),
		Key:    aws.String(objectKey),
		Body:   file,
		ACL:    types.ObjectCannedACLPrivate,
	})
	if err != nil {
		log.Println("[ERROR] Upload to Backblaze B2 failed:", err)
		return
	}

	log.Println("[UPLOAD] File uploaded to B2 bucket:", objectKey)
}
