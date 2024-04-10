package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Define a WaitGroup globally to keep track of ongoing processes
var wg sync.WaitGroup

func handleShutdown() {
	// Handle SIGINT and SIGTERM (ctrl+c) for graceful shutdown
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
