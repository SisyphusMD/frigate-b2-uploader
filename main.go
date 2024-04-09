package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gorilla/websocket"
)

// Define a WaitGroup globally to keep track of ongoing processes
var wg sync.WaitGroup

// FrigateMessage represents a message received from Frigate's WebSocket
type FrigateMessage struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"` // Use RawMessage for flexible unmarshalling (It is sometimes a JSON-encoded string, sometimes a number, so need to stay generic here, and will turn back into a string later)
}

// EventPayload represents the payload for an event
type EventPayload struct {
	After EventDetails `json:"after"`
	Type  string       `json:"type"`
}

// EventDetails contains details about an event
type EventDetails struct {
	ID        string   `json:"id"`
	HasClip   bool     `json:"has_clip"`
	Label     string   `json:"label"`
	Camera    string   `json:"camera"`
	EndTime   *float64 `json:"end_time"` // Allows for null or a timestamp
	StartTime *float64 `json:"start_time"`
}

func main() {
	config := LoadConfig() // Load the configuration

	handleShutdown()

	wsURL := fmt.Sprintf("ws://%s:%s/ws", config.FrigateIPAddress, config.FrigatePort)
	dialer := websocket.DefaultDialer
	sess := newAWSSession(config.AWSRegion, config.AWSEndpoint, config.AWSAccessKeyID, config.AWSSecretAccessKey)

	mainLoop(wsURL, dialer, sess, config.FrigateIPAddress, config.FrigatePort, config.BucketName)
}

func handleShutdown() {
	// Handle SIGINT and SITERM (ctrl+c) for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutdown signal received. Waiting for uploads to complete...")
		wg.Wait() // Wait for all goroutines to finish
		log.Println("All uploads completed. Exiting now.")
		os.Exit(0)
	}()
}

func mainLoop(wsURL string, dialer *websocket.Dialer, sess *session.Session, frigateIPAddress string, frigatePort string, bucketName string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down gracefully...")
			return
		default:
			connectAndProcessMessages(ctx, wsURL, dialer, sess, frigateIPAddress, frigatePort, bucketName)
		}
	}
}
