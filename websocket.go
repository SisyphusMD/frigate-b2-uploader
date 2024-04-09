package main

// WebSocket related imports
import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gorilla/websocket"
)

func connectAndProcessMessages(ctx context.Context, wsURL string, dialer *websocket.Dialer, sess *session.Session, frigateIPAddress string, frigatePort string, bucketName string) {
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

	processMessages(ctx, conn, sess, frigateIPAddress, frigatePort, bucketName)
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
