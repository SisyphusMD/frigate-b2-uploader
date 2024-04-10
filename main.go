package main

import (
	"context"
	"log"
	"time"
)

func main() {
	config := loadConfig() // Load the configuration

	handleShutdown()

	mainLoop(config)
}

func mainLoop(config Config) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	retryDelay := 1 * time.Second    // Initial retry delay
	const maxDelay = 1 * time.Minute // Maximum delay between retries

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down gracefully...")
			return
		default:
			frigateEventBridge(config, ctx)
			time.Sleep(retryDelay)
			// Increase delay for next retry, up to a maximum
			retryDelay *= 2
			if retryDelay > maxDelay {
				retryDelay = maxDelay
			}
		}
	}
}
