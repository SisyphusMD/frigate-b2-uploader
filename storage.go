package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func uploadClip(config Config, clipURL string, objectKey string) error {
	// Future enhancement: use config to decide which function to call.
	// For now, directly call uploadClipToB2.
	err := uploadClipToB2(config.AWSRegion, config.AWSEndpoint, config.AWSAccessKeyID, config.AWSSecretAccessKey, clipURL, objectKey, config.BucketName)
	if err != nil {
		log.Printf("Failed to upload clip: %v", err)
		return err
	}
	return nil
}

// uploadClipToB2 uploads a clip to the B2 storage.
func uploadClipToB2(awsRegion string, awsEndpoint string, awsAccessKeyID string, awsSecretAccessKey string, clipURL string, objectKey string, bucketName string) error {
	// Initialize or retrieve an existing AWS session
	sess := initAWSSession(awsRegion, awsEndpoint, awsAccessKeyID, awsSecretAccessKey)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(clipURL)
	if err != nil {
		return fmt.Errorf("unable to download file: %v", err)
	}
	defer resp.Body.Close()

	svc := s3.New(sess)

	uploader := s3manager.NewUploaderWithClient(svc)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(objectKey),
		Body:        resp.Body,
		ContentType: aws.String("video/mp4"),
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %v", err)
	}

	log.Printf("Successfully uploaded %s to %s\n", objectKey, bucketName)
	return nil
}
