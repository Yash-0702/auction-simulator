package providers

import (
	"time"
)

// ConfigProvider defines the interface for application configuration.
// All config access goes through this interface (dependency injection).
type ConfigProvider interface {
	GetNumBidders() int
	GetNumAuctions() int
	GetNumAttributes() int
	GetAuctionTimeout() time.Duration
	GetMaxCPUs() int
	GetMaxMemoryMB() int
	GetResultsDir() string
}
