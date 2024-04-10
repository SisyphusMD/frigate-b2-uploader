package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

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

func processMessages(config Config, ctx context.Context, conn *websocket.Conn) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return // Return to attempt reconnection
		}
		go handleMessage(config, ctx, message)
	}
}

// handleMessage processes each message received from the WebSocket.
func handleMessage(config Config, ctx context.Context, message []byte) {
	var msg FrigateMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error unmarshalling message: %v", err)
		return
	}

	if msg.Topic != "events" {
		return // Ignore non-event messages
	}

	processEventMessage(config, ctx, msg.Payload)
}

// processEventMessage handles the logic specific to event messages
func processEventMessage(config Config, ctx context.Context, payload json.RawMessage) {
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
		go prepareClip(config, ctx, eventPayload)
	}
}

// shouldUploadClip determines if a clip should be uploaded based on the event payload.
func shouldUploadClip(payload EventPayload) bool {
	return payload.Type == "end" && payload.After.EndTime != nil && payload.After.HasClip && payload.After.Label == "person"
}

// uploadEventClip handles the uploading of event clips.
func prepareClip(config Config, ctx context.Context, payload EventPayload) {
	wg.Add(1)       // Increment the WaitGroup counter
	defer wg.Done() // Decrement the counter when the function exits

	// Log that an event has been triggered
	eventTime := time.Unix(int64(*payload.After.StartTime), 0) // Convert UNIX timestamp to time.Time

	clipURL := fmt.Sprintf("http://%s:%s/api/events/%s/clip.mp4", config.FrigateIPAddress, config.FrigatePort, payload.After.ID)

	objectKey := fmt.Sprintf("/%d/%02d/%d%02d%02d_%02d%02d%02d_%s_%s.mp4",
		eventTime.Year(), eventTime.Month(),
		eventTime.Year(), eventTime.Month(), eventTime.Day(),
		eventTime.Hour(), eventTime.Minute(), eventTime.Second(),
		payload.After.Camera, payload.After.ID)

	log.Printf("Event triggered at %s on camera %s. Waiting for clip to be ready...", eventTime.Format("2006-01-02 15:04:05"), payload.After.Camera)

	// Wait for clip to be ready or for a shutdown signal.
	select {
	case <-time.After(12 * time.Second): // Wait for clip to be ready, as per https://github.com/blakeblackshear/frigate/issues/6662, respects graceful shutdown
		log.Printf("Preparing to upload clip for event at %s on camera %s.", eventTime.Format("2006-01-02 15:04:05"), payload.After.Camera)
	case <-ctx.Done():
		log.Printf("Shutdown signal received, but proceeding with upload for clip: %s", payload.After.ID)
	}

	if err := uploadClip(config.StorageBackends, clipURL, objectKey); err != nil {
		log.Printf("Failed to upload clip: %v", err)
	}
}
