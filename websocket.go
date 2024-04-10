package main

import (
	"context"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func connectWebSocket(ctx context.Context, wsURL string) (*websocket.Conn, error) {
	for {
		select {
		case <-ctx.Done():
			// Context canceled, stop trying to connect
			return nil, ctx.Err()
		default:
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				log.Printf("Error connecting to Websocket: %s, retrying in 5 seconds...", err)
				time.Sleep(5 * time.Second) // Wait before retrying
				continue                    // Try connecting again
			}
			log.Println("Connected to WebSocket")

			// Setup ping handler as soon as connection is established
			setupPingHandler(conn)

			return conn, nil // Connection successful, return the connection
		}
	}
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
