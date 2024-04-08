package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gorilla/websocket"
)

// Configurable constants through environment variables

var (
	frigateIPAddress   = getEnv("FRIGATE_IP_ADDRESS")
	frigatePort        = getEnv("FRIGATE_PORT")
	awsRegion          = getEnv("AWS_REGION")
	awsEndpoint        = getEnv("AWS_ENDPOINT")
	awsAccessKeyID     = getEnv("AWS_ACCESS_KEY_ID")
	awsSecretAccessKey = getEnv("AWS_SECRET_ACCESS_KEY")
	bucketName         = getEnv("BUCKET_NAME")
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
	ID      string   `json:"id"`
	HasClip bool     `json:"has_clip"`
	Label   string   `json:"label"`
	EndTime *float64 `json:"end_time"` // Allows for null or a timestamp
}

// getEnv retrieves environment variable value or exits if the variable is not set.
func getEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		log.Fatalf("Environment variable %s must be set", key)
	}
	return value
}

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
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   resp.Body,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %v", err)
	}

	log.Printf("Successfully uploaded %s to %s\n", objectKey, bucketName)
	return nil
}

func main() {
	handleShutdown()

	wsURL := fmt.Sprintf("ws://%s:%s/ws", frigateIPAddress, frigatePort)
	dialer := websocket.DefaultDialer
	sess := newAWSSession()

	mainLoop(wsURL, dialer, sess)
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

func mainLoop(wsURL string, dialer *websocket.Dialer, sess *session.Session) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down gracefully...")
			return
		default:
			connectAndProcessMessages(ctx, wsURL, dialer, sess)
		}
	}
}

func connectAndProcessMessages(ctx context.Context, wsURL string, dialer *websocket.Dialer, sess *session.Session) {
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		log.Printf("Error connecting to WebSocket: %s, retrying...", err)
		time.Sleep(5 * time.Second)
		return
	}
	defer conn.Close()
	log.Println("Connected to Frigate WebSocket")

	// Setting up a ping handler - consider adjusting the ping period as necessary
	setupPingHandler(conn)

	processMessages(ctx, conn, sess)
}

func setupPingHandler(conn *websocket.Conn) {
	pingPeriod := 30 * time.Second // Interval for sending pings (must be longer than pongWait)
	pongWait := 10 * time.Second   // Time to wait for a pong response (must be shorter than pingPeriod)

	pongReceived := make(chan struct{})

	// Update lastPongReceived upon receiving a pong
	conn.SetPongHandler(func(appData string) error {
		select {
		case pongReceived <- struct{}{}:
		default:
			// Prevent blocking if no one's listening
		}
		return nil
	})

	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()

		for range ticker.C {
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping error: %s, connection will be retried...", err)
				return
			}

			pongTimer := time.NewTimer(pongWait)

			select {
			case <-pongReceived:
				pongTimer.Stop() // Stop the timer when a pong is received. No need to drain since we don't reuse it.
			case <-pongTimer.C:
				log.Println("Pong not received within expected timeframe.") // Pong wait timer expired without receiving a pong
			}
		}
	}()
}

func processMessages(ctx context.Context, conn *websocket.Conn, sess *session.Session) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return // Return to attempt reconnection
		}
		go handleMessage(ctx, sess, message)
	}
}

// handleMessage processes each message received from the WebSocket.
func handleMessage(ctx context.Context, sess *session.Session, message []byte) {
	var msg FrigateMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error unmarshalling message: %v", err)
		return
	}

	if msg.Topic != "events" {
		return // Ignore non-event messages
	}

	processEventMessage(ctx, sess, msg.Payload)
}

// processEventMessage handles the logic specific to event messages
func processEventMessage(ctx context.Context, sess *session.Session, payload json.RawMessage) {
	// Now, we have to unmarshal the payload into a string because it's a JSON-encoded string when the topic is "events", but it is sometimes a number on other topics.
	var payloadStr string
	if err := json.Unmarshal(payload, &payloadStr); err != nil {
		log.Printf("Error unmarshalling payload into string: %v", err)
		return
	}

	var eventPayload EventPayload
	if err := json.Unmarshal([]byte(payloadStr), &eventPayload); err != nil {
		log.Printf("Error unmarshalling event payload: %v", err)
		return
	}

	if shouldUploadClip(eventPayload) {
		go uploadEventClip(ctx, sess, eventPayload.After.ID)
	}
}

// shouldUploadClip determines if a clip should be uploaded based on the event payload.
func shouldUploadClip(payload EventPayload) bool {
	return payload.Type == "end" && payload.After.EndTime != nil && payload.After.HasClip && payload.After.Label == "person"
}

// uploadEventClip handles the uploading of event clips.
func uploadEventClip(ctx context.Context, sess *session.Session, clipID string) {
	wg.Add(1)       // Increment the WaitGroup counter
	defer wg.Done() // Decrement the counter when the function exits

	// Wait for clip to be ready or for a shutdown signal.
	select {
	case <-time.After(12 * time.Second): // Wait for clip to be ready, as per https://github.com/blakeblackshear/frigate/issues/6662, respects graceful shutdown
	case <-ctx.Done():
		// The context was cancelled, but we'll log and continue with the upload to ensure all initiated operations complete.
		log.Printf("Shutdown signal received, but proceeding with upload for clip: %s", clipID)
	}

	clipURL := fmt.Sprintf("http://%s:%s/api/events/%s/clip.mp4", frigateIPAddress, frigatePort, clipID)
	log.Printf("Preparing to upload clip: %s", clipURL)

	objectKey := fmt.Sprintf("%s.mp4", clipID)
	if err := uploadClipToB2(sess, clipURL, objectKey); err != nil {
		log.Printf("Failed to upload clip: %v", err)
	}
}
