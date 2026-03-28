package models

import "time"

// Bid represents a single bid from a bidder.
type Bid struct {
	BidderID  int       `json:"bidder_id"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}
