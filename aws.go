package main

import (
	"log"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

var (
	awsSession *session.Session
	once       sync.Once
)

// initAWSSession initializes a new AWS session on its first call and returns the same session on subsequent calls.
func initAWSSession(awsRegion string, awsEndpoint string, awsAccessKeyId string, awsSecretAccessKey string) *session.Session {
	once.Do(func() {
		var err error
		awsSession, err = session.NewSession(&aws.Config{
			Region:      aws.String(awsRegion),
			Endpoint:    aws.String(awsEndpoint),
			Credentials: credentials.NewStaticCredentials(awsAccessKeyId, awsSecretAccessKey, ""),
		})
		if err != nil {
			log.Fatalf("Failed to create AWS session: %s", err)
		}
	})
	return awsSession
}
