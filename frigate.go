package main

import (
	"context"
	"fmt"
	"log"
)

func frigateEventBridge(config Config, ctx context.Context) {
	wsURL := fmt.Sprintf("ws://%s:%s/ws", config.FrigateIPAddress, config.FrigatePort)
	conn, err := connectWebSocket(ctx, wsURL) // Capture the connection and error
	if err != nil {
		log.Printf("Failed to connect to WebSocket: %v", err)
		return // Exit if connection fails
	}
	defer conn.Close() // Ensure the connection is closed when this function exits

	processMessages(config, ctx, conn)
}
