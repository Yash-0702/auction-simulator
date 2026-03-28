package server

import (
	"fmt"

	"auction-simulator/providers/configprovider"
	"auction-simulator/services/auction"
)

// Server holds all initialized providers and handlers.
// Acts as the dependency injection container for the application.
type Server struct {
	Config         *configprovider.ConfigProvider
	AuctionHandler *auction.Handler
}

// Stop performs graceful cleanup of all resources.
func (s *Server) Stop() {
	fmt.Println("Server stopped gracefully")
}

// SrvInit initializes all providers and wires dependencies.
// This is the single place where all components are created and connected.
func SrvInit() *Server {
	// Initialize config provider (loads .env file, reads env vars, sets defaults)
	config := configprovider.NewConfigProvider()

	// Apply resource limits (GOMAXPROCS for vCPU, GOMEMLIMIT for RAM)
	config.ApplyResourceLimits()

	// Create auction handler with config injected via interface
	auctionHandler := auction.NewHandler(config)

	return &Server{
		Config:         config,
		AuctionHandler: auctionHandler,
	}
}
