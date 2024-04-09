package main

// AWS related imports
import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// newAWSSession initializes a new AWS session.
func newAWSSession() *session.Session {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(awsRegion),
		Endpoint:    aws.String(awsEndpoint),
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, ""),
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %s", err)
	}
	return sess
}

// uploadClipToB2 uploads a clip to the B2 storage.
func uploadClipToB2(sess *session.Session, clipURL, objectKey string) error {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

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
		return fmt.Errorf("failed to upload file: %v", err)
	}

	log.Printf("Successfully uploaded %s to %s\n", objectKey, bucketName)
	return nil
}
