package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"auction-simulator/server"
)

func main() {
	// Initialize all providers and wire dependencies
	srv := server.SrvInit()

	// Listen for OS signals (Ctrl+C or kill) for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\n\nReceived signal: %s — shutting down gracefully...\n", sig)
		fmt.Println("Waiting for in-progress auctions to finish (no new auctions will start)")
		cancel() // cancel context to stop launching new auctions
	}()

	// Execute all auctions concurrently
	srv.AuctionHandler.RunAllAuctions(ctx)

	// Cleanup
	srv.Stop()
}
