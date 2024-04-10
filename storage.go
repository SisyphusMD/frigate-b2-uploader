package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func uploadClip(storageBackends string, clipURL string, objectKey string) error {
	// Check if B2 in storageBackends
	if strings.Contains(storageBackends, "B2") {
		b2Config := loadB2Config()
		err := uploadClipToB2(b2Config, clipURL, objectKey)
		if err != nil {
			log.Printf("Failed to upload clip to B2: %v", err)
			return err
		}
		return nil // Successfully uploaded to B2
	}
	// If reaching this point, no supported storage backend was found or specified
	// You might want to return a specific error indicating that
	return fmt.Errorf("no supported storage backend found in storageBackends: %s", storageBackends)
}

// uploadClipToB2 uploads a clip to the B2 storage.
func uploadClipToB2(b2Config B2Config, clipURL string, objectKey string) error {
	// Initialize or retrieve an existing AWS session
	sess := initAWSSession(b2Config.Region, b2Config.Endpoint, b2Config.AccessKeyID, b2Config.SecretAccessKey)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(clipURL)
	if err != nil {
		return fmt.Errorf("unable to download file: %v", err)
	}
	defer resp.Body.Close()

	svc := s3.New(sess)

	uploader := s3manager.NewUploaderWithClient(svc)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(b2Config.BucketName),
		Key:         aws.String(objectKey),
		Body:        resp.Body,
		ContentType: aws.String("video/mp4"),
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %v", err)
	}

	log.Printf("Successfully uploaded %s to %s\n", objectKey, b2Config.BucketName)
	return nil
}
