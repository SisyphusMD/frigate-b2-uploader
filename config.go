package main

import (
	"log"
	"os"
)

type Config struct {
	FrigateIPAddress   string
	FrigatePort        string
	AWSRegion          string
	AWSEndpoint        string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	BucketName         string
}

func LoadConfig() Config {
	return Config{
		FrigateIPAddress:   getEnv("FRIGATE_IP_ADDRESS"),
		FrigatePort:        getEnv("FRIGATE_PORT"),
		AWSRegion:          getEnv("AWS_REGION"),
		AWSEndpoint:        getEnv("AWS_ENDPOINT"),
		AWSAccessKeyID:     getEnv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY"),
		BucketName:         getEnv("BUCKET_NAME"),
	}
}

// getEnv retrieves environment variable value or exits if the variable is not set.
func getEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		log.Fatalf("Environment variable %s must be set", key)
	}
	return value
}
