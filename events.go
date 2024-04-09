package main

// Event processing related imports
import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gorilla/websocket"
)

func processMessages(ctx context.Context, conn *websocket.Conn, sess *session.Session, frigateIPAddress string, frigatePort string, bucketName string) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return // Return to attempt reconnection
		}
		go handleMessage(ctx, sess, message, frigateIPAddress, frigatePort, bucketName)
	}
}

// handleMessage processes each message received from the WebSocket.
func handleMessage(ctx context.Context, sess *session.Session, message []byte, frigateIPAddress string, frigatePort string, bucketName string) {
	var msg FrigateMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error unmarshalling message: %v", err)
		return
	}

	if msg.Topic != "events" {
		return // Ignore non-event messages
	}

	processEventMessage(ctx, sess, msg.Payload, frigateIPAddress, frigatePort, bucketName)
}

// processEventMessage handles the logic specific to event messages
func processEventMessage(ctx context.Context, sess *session.Session, payload json.RawMessage, frigateIPAddress string, frigatePort string, bucketName string) {
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
		go uploadEventClip(ctx, sess, eventPayload, frigateIPAddress, frigatePort, bucketName)
	}
}

// shouldUploadClip determines if a clip should be uploaded based on the event payload.
func shouldUploadClip(payload EventPayload) bool {
	return payload.Type == "end" && payload.After.EndTime != nil && payload.After.HasClip && payload.After.Label == "person"
}

// uploadEventClip handles the uploading of event clips.
func uploadEventClip(ctx context.Context, sess *session.Session, payload EventPayload, frigateIPAddress string, frigatePort string, bucketName string) {
	wg.Add(1)       // Increment the WaitGroup counter
	defer wg.Done() // Decrement the counter when the function exits

	// Log that an event has been triggered
	eventTime := time.Unix(int64(*payload.After.StartTime), 0) // Convert UNIX timestamp to time.Time
	log.Printf("Event triggered at %s on camera %s. Waiting for clip to be ready...", eventTime.Format("2006-01-02 15:04:05"), payload.After.Camera)

	// Wait for clip to be ready or for a shutdown signal.
	select {
	case <-time.After(12 * time.Second): // Wait for clip to be ready, as per https://github.com/blakeblackshear/frigate/issues/6662, respects graceful shutdown
		// Log that we are now preparing to upload the clip, after the wait
		log.Printf("Preparing to upload clip for event at %s on camera %s.", eventTime.Format("2006-01-02 15:04:05"), payload.After.Camera)
	case <-ctx.Done():
		// The context was cancelled, but we'll log and continue with the upload to ensure all initiated operations complete.
		log.Printf("Shutdown signal received, but proceeding with upload for clip: %s", payload.After.ID)
	}

	clipURL := fmt.Sprintf("http://%s:%s/api/events/%s/clip.mp4", frigateIPAddress, frigatePort, payload.After.ID)

	objectKey := fmt.Sprintf("/%d/%02d/%d%02d%02d_%02d%02d%02d_%s_%s.mp4",
		eventTime.Year(), eventTime.Month(),
		eventTime.Year(), eventTime.Month(), eventTime.Day(),
		eventTime.Hour(), eventTime.Minute(), eventTime.Second(),
		payload.After.Camera, payload.After.ID)

	if err := uploadClipToB2(sess, clipURL, objectKey, bucketName); err != nil {
		log.Printf("Failed to upload clip: %v", err)
	}
}
