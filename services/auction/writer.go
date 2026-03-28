package auction

import (
	"fmt"
	"os"
	"time"

	"auction-simulator/models"
	"auction-simulator/providers"
	"auction-simulator/utils"
)

// WriteResults writes per-auction JSON files and a summary file.
func WriteResults(cfg providers.ConfigProvider, dir string, results []models.AuctionResult, totalDuration time.Duration) {
	// Clear previous results to avoid stale files from prior runs
	if _, err := os.Stat(dir); err == nil {
		fmt.Printf("\nClearing previous results in '%s/'...\n", dir)
		os.RemoveAll(dir)
	}

	// Only write results for auctions that actually ran (AuctionID > 0)
	var completedResults []models.AuctionResult
	for _, r := range results {
		if r.AuctionID == 0 {
			continue // auction was never launched (skipped due to shutdown)
		}
		completedResults = append(completedResults, r)
		filename := fmt.Sprintf("auction_%02d.json", r.AuctionID)
		if err := utils.WriteJSON(dir, filename, r); err != nil {
			fmt.Printf("Error writing result for auction %d: %v\n", r.AuctionID, err)
		}
	}

	summary := models.Summary{
		Config: models.SummaryConfig{
			NumBidders:     cfg.GetNumBidders(),
			NumAuctions:    cfg.GetNumAuctions(),
			AuctionTimeout: cfg.GetAuctionTimeout().String(),
			MaxCPUs:        cfg.GetMaxCPUs(),
			MaxMemoryMB:    cfg.GetMaxMemoryMB(),
		},
		TotalAuctions: len(completedResults),
		TotalDuration: totalDuration.String(),
		Results:       completedResults,
	}
	if err := utils.WriteJSON(dir, "summary.json", summary); err != nil {
		fmt.Printf("Error writing summary: %v\n", err)
	}

	fmt.Printf("\nResults written to ./%s/\n", dir)
}
