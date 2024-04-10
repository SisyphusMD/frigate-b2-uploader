package main

import (
	"log"
	"os"
)

type Config struct {
	FrigateIPAddress string
	FrigatePort      string
	StorageBackends  string
}

type B2Config struct {
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
}

func loadConfig() Config {
	return Config{
		FrigateIPAddress: getEnv("FRIGATE_IP_ADDRESS"),
		FrigatePort:      getEnv("FRIGATE_PORT"),
		StorageBackends:  getEnv("STORAGE_BACKENDS"),
	}
}

func loadB2Config() B2Config {
	return B2Config{
		Region:          getEnv("B2_REGION"),
		Endpoint:        getEnv("B2_ENDPOINT"),
		AccessKeyID:     getEnv("B2_ACCESS_KEY_ID"),
		SecretAccessKey: getEnv("B2_SECRET_ACCESS_KEY"),
		BucketName:      getEnv("B2_BUCKET_NAME"),
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
