package configprovider

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

// ConfigProvider implements providers.ConfigProvider interface.
type ConfigProvider struct {
	NumBidders     int
	NumAuctions    int
	NumAttributes  int
	AuctionTimeout time.Duration
	MaxCPUs        int
	MaxMemoryMB    int
	ResultsDir     string
}

// NewConfigProvider creates a new config provider, loading from .env and env vars.
func NewConfigProvider() *ConfigProvider {
	// Load .env file if present (env vars from system/command line always work)
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using defaults and system environment variables")
	} else {
		fmt.Println("Loaded configuration from .env file (system env vars take priority)")
	}

	cfg := &ConfigProvider{
		NumBidders:     100,
		NumAuctions:    40,
		NumAttributes:  20,
		AuctionTimeout: 2 * time.Second,
		MaxCPUs:        runtime.NumCPU(),
		MaxMemoryMB:    512,
		ResultsDir:     "results",
	}

	if v := os.Getenv("NUM_BIDDERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.NumBidders = n
		}
	}
	if v := os.Getenv("NUM_AUCTIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.NumAuctions = n
		}
	}
	if v := os.Getenv("AUCTION_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.AuctionTimeout = d
		}
	}
	if v := os.Getenv("MAX_CPUS"); v != "" {
		if n, err := strconv.Atoi(v); err != nil || n < 1 {
			fmt.Printf("Warning: MAX_CPUS must be a whole number >= 1 (got %q), using default: %d\n", v, cfg.MaxCPUs)
		} else {
			if n > runtime.NumCPU() {
				fmt.Printf("Notice: MAX_CPUS=%d exceeds available CPUs (%d). This is allowed but won't improve performance beyond %d.\n", n, runtime.NumCPU(), runtime.NumCPU())
			}
			cfg.MaxCPUs = n
		}
	}
	if v := os.Getenv("MAX_MEMORY_MB"); v != "" {
		if n, err := strconv.Atoi(v); err != nil || n < 1 {
			fmt.Printf("Warning: MAX_MEMORY_MB must be a whole number >= 1 (got %q), using default: %d\n", v, cfg.MaxMemoryMB)
		} else {
			systemMemMB := int(getSystemMemoryMB())
			if systemMemMB > 0 && n > systemMemMB {
				fmt.Printf("Notice: MAX_MEMORY_MB=%d exceeds available system RAM (%d MB). This is allowed but may cause excessive swapping.\n", n, systemMemMB)
			}
			cfg.MaxMemoryMB = n
		}
	}
	if v := os.Getenv("RESULTS_DIR"); v != "" {
		cfg.ResultsDir = v
	}

	return cfg
}

// getSystemMemoryMB returns total system RAM in MB. Returns 0 if unable to detect.
func getSystemMemoryMB() uint64 {
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		return 0
	}
	return info.Totalram / 1024 / 1024
}

// ApplyResourceLimits sets GOMAXPROCS and GOMEMLIMIT.
func (c *ConfigProvider) ApplyResourceLimits() {
	runtime.GOMAXPROCS(c.MaxCPUs)
	debug.SetMemoryLimit(int64(c.MaxMemoryMB) * 1024 * 1024)
}

// Interface method implementations
func (c *ConfigProvider) GetNumBidders() int            { return c.NumBidders }
func (c *ConfigProvider) GetNumAuctions() int           { return c.NumAuctions }
func (c *ConfigProvider) GetNumAttributes() int         { return c.NumAttributes }
func (c *ConfigProvider) GetAuctionTimeout() time.Duration { return c.AuctionTimeout }
func (c *ConfigProvider) GetMaxCPUs() int               { return c.MaxCPUs }
func (c *ConfigProvider) GetMaxMemoryMB() int           { return c.MaxMemoryMB }
func (c *ConfigProvider) GetResultsDir() string         { return c.ResultsDir }
