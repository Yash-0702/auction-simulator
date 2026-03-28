package models

import "time"

// AuctionResult holds the outcome of a single auction.
type AuctionResult struct {
	AuctionID    int           `json:"auction_id"`
	Item         Item          `json:"item"`
	TotalBidders int           `json:"total_bidders"`
	BidsReceived int           `json:"bids_received"`
	Bids         []Bid         `json:"bids"`
	Winner       *Bid          `json:"winner"`
	Duration     time.Duration `json:"duration_ns"`
	DurationStr  string        `json:"duration"`
	TimedOut     bool          `json:"timed_out"`
}

// SummaryConfig holds the configuration used for this run.
type SummaryConfig struct {
	NumBidders     int    `json:"num_bidders"`
	NumAuctions    int    `json:"num_auctions"`
	AuctionTimeout string `json:"auction_timeout"`
	MaxCPUs        int    `json:"max_cpus"`
	MaxMemoryMB    int    `json:"max_memory_mb"`
}

// Summary holds the aggregated results of all auctions.
type Summary struct {
	Config        SummaryConfig    `json:"config"`
	TotalAuctions int              `json:"total_auctions"`
	TotalDuration string           `json:"total_duration"`
	Results       []AuctionResult  `json:"results"`
}
