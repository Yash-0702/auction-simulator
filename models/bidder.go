package models

import "math/rand"

// Bidder represents a participant with preferences for item attributes.
type Bidder struct {
	ID      int
	Weights [20]float64
	Budget  float64
}

// NewBidders creates n bidders with randomized preference weights and budgets.
func NewBidders(n int, rng *rand.Rand) []Bidder {
	bidders := make([]Bidder, n)
	for i := 0; i < n; i++ {
		var weights [20]float64
		for j := 0; j < 20; j++ {
			weights[j] = rng.Float64()
		}
		bidders[i] = Bidder{
			ID:      i + 1,
			Weights: weights,
			Budget:  80000 + rng.Float64()*20000,
		}
	}
	return bidders
}
