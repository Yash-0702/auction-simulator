package auction

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"auction-simulator/models"
	"auction-simulator/providers"
)

// Service handles the auction business logic.
type Service struct {
	config providers.ConfigProvider
}

// NewService creates a new auction service with injected dependencies.
func NewService(config providers.ConfigProvider) *Service {
	return &Service{config: config}
}

// EvaluateAndBid computes a bid for the given item. Returns nil if the bidder
// chooses not to bid (simulated by a random skip chance).
func (s *Service) evaluateAndBid(b models.Bidder, item models.Item, rng *rand.Rand) *models.Bid {
	// 20% chance the bidder doesn't respond
	if rng.Float64() < 0.20 {
		return nil
	}

	// Compute weighted score from item attributes
	attrValues := item.AttributeValues()
	var score float64
	for i := 0; i < 20; i++ {
		score += b.Weights[i] * attrValues[i]
	}

	// Scale score to a bid amount (normalize: max possible score is 20)
	bidAmount := (score / 20.0) * b.Budget

	// Add some randomness (+/- 10%)
	jitter := 0.9 + rng.Float64()*0.2
	bidAmount *= jitter

	if bidAmount <= 0 {
		return nil
	}

	// Simulate thinking time (0-100ms)
	time.Sleep(time.Duration(rng.Intn(100)) * time.Millisecond)

	return &models.Bid{
		BidderID:  b.ID,
		Amount:    bidAmount,
		Timestamp: time.Now(),
	}
}

// RunAuction executes a single auction:
//  1. Sends item attributes to all bidders (each in its own goroutine)
//  2. Collects bids via a channel until timeout or all bidders respond
//  3. Declares the highest bidder as winner
func (s *Service) RunAuction(ctx context.Context, auctionID int, item models.Item, bidders []models.Bidder) models.AuctionResult {
	start := time.Now()

	// Buffered channel to collect bids — bidders send, auction receives
	bidsChan := make(chan models.Bid, len(bidders))
	var wg sync.WaitGroup

	// Launch one goroutine per bidder — all evaluate the item concurrently
	for _, b := range bidders {
		wg.Add(1)
		go func(b models.Bidder) {
			defer wg.Done()
			// Each goroutine gets its own RNG to avoid lock contention
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(b.ID*auctionID)))
			bid := s.evaluateAndBid(b, item, rng)
			if bid != nil {
				select {
				case bidsChan <- *bid: // send bid to channel
				case <-ctx.Done(): // timeout already fired, stop
					return
				}
			}
		}(b)
	}

	// Close channel once all bidders are done (signals "no more bids coming")
	go func() {
		wg.Wait()
		close(bidsChan)
	}()

	// Collect bids until either: all bidders responded OR timeout fires
	var bids []models.Bid
	timedOut := false

Loop:
	for {
		select {
		case bid, ok := <-bidsChan:
			if !ok {
				break Loop // channel closed — all bidders done
			}
			bids = append(bids, bid)
		case <-ctx.Done():
			timedOut = true // timeout reached — stop collecting
			break Loop
		}
	}

	// Winner = highest bid amount
	var winner *models.Bid
	for i := range bids {
		if winner == nil || bids[i].Amount > winner.Amount {
			winner = &bids[i]
		}
	}

	duration := time.Since(start)

	return models.AuctionResult{
		AuctionID:    auctionID,
		Item:         item,
		TotalBidders: len(bidders),
		BidsReceived: len(bids),
		Bids:         bids,
		Winner:       winner,
		Duration:     duration,
		DurationStr:  duration.String(),
		TimedOut:     timedOut,
	}
}
