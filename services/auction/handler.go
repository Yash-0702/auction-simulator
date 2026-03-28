package auction

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"auction-simulator/models"
	"auction-simulator/providers"
	"auction-simulator/utils"
)

// Handler orchestrates auction execution — creates bidders, generates items,
// launches concurrent auctions with resource throttling, and writes results.
type Handler struct {
	service *Service
	config  providers.ConfigProvider
}

// NewHandler creates a new auction handler with injected dependencies.
func NewHandler(config providers.ConfigProvider) *Handler {
	return &Handler{
		service: NewService(config),
		config:  config,
	}
}

// RunAllAuctions executes all auctions concurrently with semaphore-based throttling.
// Accepts a context for graceful shutdown — if cancelled, no new auctions will start
// but in-progress auctions will finish.
func (h *Handler) RunAllAuctions(ctx context.Context) {
	cfg := h.config

	fmt.Println("=== Auction Simulator ===")
	fmt.Printf("Bidders:        %d\n", cfg.GetNumBidders())
	fmt.Printf("Auctions:       %d\n", cfg.GetNumAuctions())
	fmt.Printf("Attributes:     %d\n", cfg.GetNumAttributes())
	fmt.Printf("Timeout:        %s\n", cfg.GetAuctionTimeout())
	fmt.Printf("vCPUs (GOMAXPROCS): %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("Max Memory:     %d MB\n", cfg.GetMaxMemoryMB())
	fmt.Println()

	// Create bidders
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	bidders := models.NewBidders(cfg.GetNumBidders(), rng)

	// Generate items for each auction
	numAuctions := cfg.GetNumAuctions()
	items := make([]models.Item, numAuctions)
	for i := 0; i < numAuctions; i++ {
		items[i] = models.NewItem(i+1, rng)
	}

	// Semaphore-based throttling: only MaxCPUs auctions run concurrently.
	// If MaxCPUs=4 and NumAuctions=40, auctions run in batches of 4.
	// Remaining auctions wait for a slot, ensuring resource usage scales with available vCPUs.
	results := make([]models.AuctionResult, numAuctions)
	var wg sync.WaitGroup
	sem := make(chan struct{}, cfg.GetMaxCPUs()) // buffered channel acts as semaphore

	fmt.Printf("Concurrency limit: %d (based on available vCPUs)\n", cfg.GetMaxCPUs())
	fmt.Println("Starting all auctions...")

	// Time measurement: from the start of the first auction to the completion of the last
	totalStart := time.Now()

	var completed int32
	for i := 0; i < numAuctions; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Acquire a vCPU slot — blocks if all slots are in use
			// Uses select so we can also respond to shutdown signal while waiting
			select {
			case sem <- struct{}{}:
				// got a slot
			case <-ctx.Done():
				// shutdown requested while waiting for a slot — skip this auction
				return
			}
			defer func() { <-sem }() // release the slot when this auction completes

			// Double-check context after acquiring slot
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Each auction gets its own timeout context
			auctionCtx, cancel := context.WithTimeout(context.Background(), cfg.GetAuctionTimeout())
			defer cancel()

			// results[idx] is safe: each goroutine writes to a unique index
			results[idx] = h.service.RunAuction(auctionCtx, idx+1, items[idx], bidders)
			atomic.AddInt32(&completed, 1)
		}(i)
	}

	// Block until all auction goroutines complete (either finished or skipped)
	wg.Wait()
	totalDuration := time.Since(totalStart)
	launched := int(atomic.LoadInt32(&completed))

	// Print results
	fmt.Println()
	if launched < numAuctions {
		fmt.Printf("=== %d/%d Auctions Completed (graceful shutdown) ===\n", launched, numAuctions)
	} else {
		fmt.Println("=== All Auctions Complete ===")
	}
	fmt.Printf("Total time (first start → last complete): %s\n", totalDuration)

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	fmt.Printf("Memory in use:  %.2f MB (currently allocated and not yet freed)\n", float64(memStats.Alloc)/1024/1024)
	fmt.Printf("Total alloc:    %.2f MB (cumulative allocations over program lifetime, includes freed memory)\n", float64(memStats.TotalAlloc)/1024/1024)
	fmt.Printf("Memory limit:   %d MB (max allowed by GOMEMLIMIT)\n", cfg.GetMaxMemoryMB())
	fmt.Printf("GC cycles:      %d (garbage collector ran this many times to stay within limit)\n", memStats.NumGC)
	fmt.Printf("Auction goroutines remaining: 0 (all %d workers cleaned up)\n", launched)

	// Write output files
	h.writeResults(cfg.GetResultsDir(), results, totalDuration)
}

// writeResults writes per-auction JSON files and a summary file.
func (h *Handler) writeResults(dir string, results []models.AuctionResult, totalDuration time.Duration) {
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

	cfg := h.config
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
