# Auction Simulator

A concurrent auction simulator built in Go that runs multiple auctions simultaneously with simulated bidders. Follows a layered, service-oriented architecture with dependency injection.

## Live CI/CD Demo

> **[View the CI/CD pipeline run on GitHub Actions](https://github.com/Yash-0702/auction-simulator/actions/runs/23689245550)** — includes build logs, server output, race detection results, and downloadable auction artifacts across multiple resource configurations.

## Features

- **100 bidders** with randomized attribute preferences
- **40 auctions** launched via goroutines, throttled to `NumCPU` concurrent slots
- **20 attributes** per item used for bid evaluation
- **Configurable timeout** per auction using `context.WithTimeout`
- **Resource standardization** via semaphore-based throttling and `GOMEMLIMIT`
- **Per-auction output files** in JSON format
- **Timing measurement** from first auction start to last completion
- **Environment variable configuration** via `.env` file (powered by `godotenv`)
- **Graceful shutdown** — Ctrl+C stops new auctions, waits for in-progress ones to finish, writes partial results

## Prerequisites

- Go 1.26.1 or higher (for local builds)
- Docker (for containerized builds)

## Build & Run

### Using Docker

```bash
docker build -t auction-simulator .
docker run --rm --env-file .env auction-simulator
```

Override settings inline:
```bash
docker run --rm -e NUM_AUCTIONS=10 -e NUM_BIDDERS=50 auction-simulator
```

Persist results to the host:
```bash
docker run --rm --env-file .env -v ./results:/app/results auction-simulator
```

### Using Go directly

```bash
go build -o auction-simulator .
./auction-simulator
```

Or run directly without building:
```bash
go run main.go
```

## Configuration

All settings can be overridden via environment variables in `.env` file (no recompilation needed):

| Env Variable      | Default          | Description                              |
|-------------------|------------------|------------------------------------------|
| `NUM_BIDDERS`     | 100              | Number of participating bidders          |
| `NUM_AUCTIONS`    | 40               | Total number of auctions to run          |
| `AUCTION_TIMEOUT` | 2s               | Timeout per auction (e.g., `500ms`, `3s`)|
| `MAX_CPUS`        | runtime.NumCPU() | Concurrent auction limit (semaphore size)|
| `MAX_MEMORY_MB`   | 512              | Memory limit for Go runtime              |
| `RESULTS_DIR`     | results          | Output directory for JSON files          |

**Examples:**

```bash
# Run with 4 vCPUs and 5s timeout
MAX_CPUS=4 AUCTION_TIMEOUT=5s ./auction-simulator

# Run with fewer bidders and auctions
NUM_BIDDERS=50 NUM_AUCTIONS=10 ./auction-simulator

# Custom output directory
RESULTS_DIR=output ./auction-simulator
```

The `.env` file is optional. If not present, defaults are used. Command-line env vars override `.env` values.

---

## How Everything Works

### Step 1: Startup and Dependency Injection

When the program starts, `main.go` calls `server.SrvInit()` which:

1. **Loads configuration** — reads `.env` file (if present) using `godotenv`, then checks environment variables, falls back to hardcoded defaults
2. **Applies resource limits** — calls `runtime.GOMAXPROCS(MaxCPUs)` to limit CPU threads and `debug.SetMemoryLimit(MaxMemoryMB)` to cap RAM usage
3. **Creates the auction handler** — injects the config via an interface (`providers.ConfigProvider`), so the handler never knows where config values come from

`main.go` also sets up a **signal listener** for graceful shutdown (explained below).

### Step 2: Creating Bidders

100 bidders are created, each with:

- **20 random preference weights** (0.0 to 1.0) — represents how much each bidder values each attribute. For example, Bidder #1 might have weight 0.95 for attribute "quality" but 0.10 for "color"
- **A random budget** ($500 to $2000) — the maximum amount they're willing to spend

Every bidder is unique — different preferences and different budgets.

### Step 3: Generating Auction Items

40 items are created, each with:

- **An ID and name** (Item-1, Item-2, ... Item-40)
- **20 named attribute values** (0 to 100) — quality, rarity, condition, age, durability, aesthetics, brand_value, market_demand, authenticity, craftsmanship, material_grade, color_vibrancy, size, portability, warranty_score, eco_friendliness, innovation, cultural_value, resale_potential, weight

### Step 4: Running 40 Auctions Concurrently

All 40 auctions are launched as goroutines (lightweight threads), but they are **throttled by a semaphore** — a buffered channel sized to `MaxCPUs`:

```
sem := make(chan struct{}, MaxCPUs)   // e.g., capacity 4

// Before each auction:
sem <- struct{}{}        // acquire slot (blocks if all 4 slots are full)

// After each auction:
<-sem                    // release slot (next waiting auction can start)
```

**Example with MAX_CPUS=4:**
```
Time 0ms:     Auctions 1,2,3,4 acquire slots → start running
              Auctions 5-40 are blocked, waiting for a slot
Time ~100ms:  Auction 1 finishes → releases slot → Auction 5 starts
Time ~100ms:  Auction 3 finishes → releases slot → Auction 6 starts
... continues until all 40 complete
```

This ensures the program never runs more auctions than available CPUs, regardless of how many total auctions there are.

### Step 5: Inside Each Auction

For each auction, the following happens:

**5a. Send item to all 100 bidders (concurrently)**

Each bidder runs in its own goroutine. They all evaluate the item at the same time. Each goroutine gets its own random number generator to avoid lock contention.

**5b. Bidder evaluates the item**

Each bidder decides whether and how much to bid:

1. **20% chance of not responding** — `if random < 0.20 → return nil`. This simulates real-world behavior where not every bidder participates. On average, ~80 out of 100 bidders respond.

2. **Calculate bid amount using weighted score:**
   ```
   score = weight[0] × attribute[0] + weight[1] × attribute[1] + ... + weight[19] × attribute[19]
   bidAmount = (score / 20) × budget
   ```
   A bidder who values what the item offers (high weights matching high attributes) will bid more. The score is normalized by dividing by 20 (number of attributes), then scaled by the bidder's budget.

3. **Add ±10% randomness (jitter)** — multiplies the bid by a random factor between 0.9 and 1.1 to simulate real-world imprecision.

4. **Simulate thinking time** — `time.Sleep(0 to 99ms)`. Each bidder takes a different amount of time to respond. This is important because slow bidders might miss the auction timeout.

5. **Submit bid** — sends the bid (bidder ID, amount, timestamp) through a Go channel to the auction collector.

**5c. Collect bids until timeout**

The auction sits in a loop with a `select` statement:

```go
select {
case bid := <-bidsChan:    // a bid arrived → add to list
case <-ctx.Done():         // timeout fired → stop collecting
}
```

Whichever happens first wins. If the 2-second timeout fires before all bidders respond, the auction closes immediately with whatever bids it has.

**5d. Declare winner**

After collecting stops, the auction scans all received bids and picks the **highest amount** as the winner. If no bids were received (all bidders skipped or were too slow), the winner is `nil`.

### Step 6: Time Measurement

The total time is measured from **before the first auction goroutine launches** to **after the last one completes**:

```go
totalStart := time.Now()        // start clock
// ... launch all auctions ...
wg.Wait()                       // wait for all to finish
totalDuration := time.Since(totalStart)  // stop clock
```

This gives the exact time from "first auction start" to "last auction complete" as required by the assignment.

### Step 7: Resource Reporting

After all auctions complete, the program reports:

| Metric | What it means |
|--------|---------------|
| Memory in use | Bytes currently allocated (live objects) |
| Total alloc | Cumulative bytes allocated over entire program lifetime (includes freed memory) |
| Memory limit | The GOMEMLIMIT cap that was set |
| GC cycles | How many times garbage collector ran to stay within memory limit |
| Auction goroutines | Should be 0 — confirms all workers cleaned up properly |

### Step 8: Writing Results

1. **Clears previous results** — deletes the `results/` directory to avoid stale files from prior runs
2. **Writes one JSON file per auction** — `auction_01.json` through `auction_40.json`, each containing the item, all bids, winner, duration, and timeout status
3. **Writes summary.json** — aggregated results with total auction count and total duration
4. The summary's `total_auctions` count always matches the number of auction JSON files

---

## Resource Standardization

The assignment requires standardizing resources with respect to vCPU and RAM. This is achieved through three mechanisms:

### vCPU Control

**1. Semaphore throttling** (`handler.go`):

A buffered channel sized to `runtime.NumCPU()` acts as a semaphore. Before an auction runs, it must acquire a slot. If all slots are taken, the auction waits. This ensures only `MaxCPUs` auctions execute concurrently:

```
MAX_CPUS=2  → 20 batches × ~100ms = ~2000ms total
MAX_CPUS=4  → 10 batches × ~100ms = ~1000ms total
MAX_CPUS=8  →  5 batches × ~100ms =  ~500ms total
MAX_CPUS=12 →  4 batches × ~100ms =  ~400ms total
```

The program automatically adapts to the machine — same code, different performance based on available CPUs.

**2. GOMAXPROCS** (`configprovider.go`):

`runtime.GOMAXPROCS(MaxCPUs)` limits how many OS threads Go uses to run goroutines. This is a secondary control alongside the semaphore.

### RAM Control

**3. GOMEMLIMIT** (`configprovider.go`):

`debug.SetMemoryLimit(MaxMemoryMB * 1024 * 1024)` tells Go's garbage collector to keep live memory under this limit. When memory approaches the limit, GC runs more aggressively to free unused objects. Memory usage is reported after execution to verify compliance.

### Configuration

Both limits are configurable via `.env`:
```
MAX_CPUS=4          # only 4 auctions run at once
MAX_MEMORY_MB=512   # GC keeps memory under 512 MB
```

### Input Validation

Both `MAX_CPUS` and `MAX_MEMORY_MB` must be **whole numbers >= 1**. Go's `runtime.GOMAXPROCS()` and `debug.SetMemoryLimit()` only accept integers — you can't use half a CPU core or 0.5 MB of RAM in Go. Fractional or invalid values are rejected with a warning, and the default is used instead:

```
MAX_CPUS=0.22     → Warning: must be a whole number >= 1, using default: 12
MAX_MEMORY_MB=0.5 → Warning: must be a whole number >= 1, using default: 512
MAX_CPUS=abc      → Warning: must be a whole number >= 1, using default: 12
MAX_CPUS=0        → Warning: must be a whole number >= 1, using default: 12
MAX_CPUS=4        → ✓ accepted
MAX_MEMORY_MB=256 → ✓ accepted
```

If valid values **exceed system resources**, the program accepts them but shows a notice:

```
MAX_CPUS=125      → Notice: exceeds available CPUs (12). Allowed but won't improve performance beyond 12.
MAX_MEMORY_MB=50000 → Notice: exceeds available system RAM (23755 MB). Allowed but may cause excessive swapping.
```

Values are not capped because in containerized environments (Docker, Kubernetes), `runtime.NumCPU()` may report the host's cores, not the container's allocation. The user may intentionally set different values.

---

## Graceful Shutdown

The program handles `Ctrl+C` (SIGINT) and `kill` (SIGTERM) signals for clean shutdown:

1. `main.go` creates a cancellable context and listens for OS signals in a background goroutine
2. When a signal is received, the context is cancelled
3. Auction goroutines waiting for a semaphore slot see `ctx.Done()` and **exit immediately** without running
4. Auctions that already acquired a slot (in-progress) **finish normally** — they have their own separate timeout context
5. Only completed auction results are written to disk
6. `srv.Stop()` performs cleanup
7. Program exits with "Server stopped gracefully"

```
^C
Received signal: interrupt — shutting down gracefully...
Waiting for in-progress auctions to finish (no new auctions will start)
Auction #11: Winner is Bidder #5 with bid $69430.89 (87 bids received, 105ms)
Auction #10: Winner is Bidder #5 with bid $65132.86 (76 bids received, 101ms)

=== 12/40 Auctions Completed (graceful shutdown) ===
Total time (first start → last complete): 713ms
...
Clearing previous results in 'results/'...

Results written to ./results/
Server stopped gracefully
```

---

## Architecture

The project follows a **layered, service-oriented architecture** with dependency injection.

```
main.go                          → Entry point, signal handling, graceful shutdown
│
├── server/
│   └── server.go                → Initializes providers, wires dependencies, graceful stop
│
├── providers/
│   ├── providers.go             → ConfigProvider interface (contract)
│   └── configprovider/
│       └── configprovider.go    → Config implementation (.env, env vars, resource limits)
│
├── services/
│   └── auction/
│       ├── handler.go           → Orchestrates auction execution (launches goroutines, writes output)
│       └── service.go           → Business logic (bid evaluation, auction running, winner selection)
│
├── models/
│   ├── item.go                  → Item struct + NewItem()
│   ├── bid.go                   → Bid struct
│   ├── bidder.go                → Bidder struct + NewBidders()
│   └── auction.go               → AuctionResult + Summary structs
│
├── utils/
│   └── utils.go                 → Shared helpers (WriteJSON)
│
└── results/                     → Generated output (gitignored)
```

### What each layer does

| Layer | Files | Responsibility |
|-------|-------|---------------|
| **Entry point** | `main.go` | Starts server, sets up signal handling, triggers shutdown |
| **Server** | `server/server.go` | Creates and wires all providers and handlers (dependency injection container) |
| **Providers** | `providers/` | Interfaces for cross-cutting concerns. `ConfigProvider` defines what config methods exist; `configprovider/` implements them by reading `.env` + env vars |
| **Services** | `services/auction/` | `handler.go` orchestrates (creates bidders, launches auctions, writes output). `service.go` contains business logic (bid calculation, auction running, winner selection) |
| **Models** | `models/` | Data structures only. Each model in its own file. Also contains factory functions (`NewItem`, `NewBidders`) |
| **Utils** | `utils/` | Shared helpers like `WriteJSON` used across the project |

### Data Flow

```
main.go
  └── server.SrvInit()
        ├── configprovider.NewConfigProvider()    → load .env + defaults
        ├── config.ApplyResourceLimits()          → GOMAXPROCS + GOMEMLIMIT
        └── auction.NewHandler(config)            → inject config into handler
              └── auction.NewService(config)      → inject config into service

  └── handler.RunAllAuctions(ctx)
        ├── models.NewBidders(100)                → create 100 bidders with random weights
        ├── models.NewItem() × 40                 → create 40 items with random attributes
        ├── semaphore throttling                  → only MaxCPUs auctions at once
        │     └── service.RunAuction()            → per auction:
        │           ├── launch 100 bidder goroutines
        │           ├── each bidder: 20% skip OR calculate weighted bid + sleep
        │           ├── collect bids via channel until timeout
        │           └── pick highest bid = winner
        ├── timing + memory reporting
        ├── clear previous results
        └── utils.WriteJSON()                     → write auction files + summary

  └── srv.Stop()                                  → graceful cleanup
```

### Design Patterns Used

| Pattern | Where | Why |
|---------|-------|-----|
| **Dependency Injection** | `providers.ConfigProvider` interface | Config access is decoupled from implementation. Handler doesn't know if config comes from `.env`, env vars, or defaults |
| **Interface-based abstraction** | `providers/providers.go` | Easy to mock for testing, easy to swap implementations |
| **Service Layer** | `services/auction/service.go` | Business logic (bid calculation, auction running) separated from orchestration |
| **Handler Layer** | `services/auction/handler.go` | Orchestration (launching goroutines, writing files) separated from business logic |
| **Factory Pattern** | `server.SrvInit()` | Single place where all components are created and connected |
| **Semaphore Pattern** | `handler.go` buffered channel | Resource-aware concurrency control tied to available vCPUs |

### Concurrency Model

The program uses these Go concurrency primitives:

| Primitive | Where | Purpose |
|-----------|-------|---------|
| **Goroutines** | `handler.go`, `service.go` | Run auctions and bidders concurrently |
| **Buffered channel (semaphore)** | `handler.go` | Limit concurrent auctions to MaxCPUs |
| **Buffered channel (bids)** | `service.go` | Bidders send bids, auction collects them |
| **sync.WaitGroup** | `handler.go`, `service.go` | Wait for all goroutines to finish |
| **context.WithTimeout** | `handler.go` | Auto-close auction after timeout |
| **context.WithCancel** | `main.go` | Cancel context on Ctrl+C for graceful shutdown |
| **select statement** | `service.go`, `handler.go` | Choose between receiving a bid OR timeout firing; acquiring semaphore OR shutdown |
| **sync/atomic** | `handler.go` | Thread-safe counter for completed auctions |

---

## Output

Each auction produces a JSON file in `results/`:

- `auction_01.json` through `auction_40.json` — individual auction results
- `summary.json` — aggregated results with total duration

### Sample JSON Output (auction_01.json)

```json
{
  "auction_id": 1,
  "item": {
    "id": 1,
    "name": "Item-1",
    "attributes": {
      "quality": 98.07,
      "rarity": 34.25,
      "condition": 7.57,
      "age": 41.46,
      "durability": 65.14,
      "aesthetics": 3.67,
      "brand_value": 72.97,
      "craftsmanship": 92.42,
      "... 20 named attributes total": 0
    }
  },
  "total_bidders": 100,
  "bids_received": 82,
  "bids": [
    {
      "bidder_id": 47,
      "amount": 1234.56,
      "timestamp": "2026-03-28T16:14:23.122778721+05:30"
    }
  ],
  "winner": {
    "bidder_id": 47,
    "amount": 1234.56,
    "timestamp": "2026-03-28T16:14:23.122778721+05:30"
  },
  "duration_ns": 105180000,
  "duration": "105.18ms",
  "timed_out": false
}
```

### Sample JSON Output (summary.json)

The summary file contains the **configuration used for the run** and **all auction results** in one file:

```json
{
  "config": {
    "num_bidders": 100,
    "num_auctions": 40,
    "auction_timeout": "2s",
    "max_cpus": 4,
    "max_memory_mb": 512
  },
  "total_auctions": 40,
  "total_duration": "1.002s",
  "results": [
    {
      "auction_id": 1,
      "item": { "..." },
      "total_bidders": 100,
      "bids_received": 82,
      "bids": [ "..." ],
      "winner": { "bidder_id": 47, "amount": 1234.56, "..." },
      "duration": "105.18ms",
      "timed_out": false
    },
    { "auction_id": 2, "..." },
    { "auction_id": 3, "..." }
  ]
}
```

| Field | What it contains |
|-------|-----------------|
| `config` | The exact settings used for this run (bidders, auctions, timeout, CPU limit, memory limit) |
| `total_auctions` | Number of auctions that completed (matches number of `auction_XX.json` files) |
| `total_duration` | Time from first auction start to last auction complete |
| `results` | Array of all individual auction results (same data as each `auction_XX.json`) |

### Sample Output (terminal)

```
Loaded configuration from .env file (system env vars take priority)
=== Auction Simulator ===
Bidders:        100
Auctions:       40
Timeout:        2s
vCPUs (GOMAXPROCS): 4
Max Memory:     512 MB

Concurrency limit: 4 (based on available vCPUs)
Starting all auctions...
Auction #1: Winner is Bidder #47 with bid $1234.56 (82 bids received, 95ms)
Auction #2: Winner is Bidder #12 with bid $987.32 (79 bids received, 88ms)
...

=== All Auctions Complete ===
Total time (first start → last complete): 1.002s
Memory in use:  0.97 MB (currently allocated and not yet freed)
Total alloc:    23.17 MB (cumulative allocations over program lifetime, includes freed memory)
Memory limit:   512 MB (max allowed by GOMEMLIMIT)
GC cycles:      9 (garbage collector ran this many times to stay within limit)
Auction goroutines remaining: 0 (all 40 workers cleaned up)

Clearing previous results in 'results/'...

Results written to ./results/
Server stopped gracefully
```

## CI/CD (GitHub Actions)

The project includes a GitHub Actions workflow (`.github/workflows/run-auction.yml`) that runs automatically on every push to `main`:

1. **Builds** the project
2. **Builds with race detector** to verify no data races
3. **Runs the simulator** with two resource configs (CPU=2/RAM=256MB and CPU=4/RAM=512MB)
4. **Verifies** 41 output files are created
5. **Writes server logs** to GitHub Summary (visible in the Actions tab)
6. **Uploads auction results** as downloadable artifacts (retained for 30 days)
7. **Cleans up** build artifacts

Can also be triggered manually from the Actions tab → "Run workflow" button.

## Verification

```bash
# Build and run with Docker
docker build -t auction-simulator .
docker run --rm -v ./results:/app/results auction-simulator

# Or build and run with Go
go build -o auction-simulator . && ./auction-simulator

# Check output files were created (should show 41: 40 auctions + 1 summary)
ls results/ | wc -l

# View a single auction result
cat results/auction_01.json

# View summary
cat results/summary.json

# Run with race detector to verify no data races
go build -race -o auction-simulator-race . && ./auction-simulator-race

# Test with different CPU limits to verify resource standardization
MAX_CPUS=2 ./auction-simulator   # ~2s with 2 CPUs
MAX_CPUS=4 ./auction-simulator   # ~1s with 4 CPUs
MAX_CPUS=8 ./auction-simulator   # ~0.5s with 8 CPUs

# Test timeout behavior (short timeout = fewer bids received)
AUCTION_TIMEOUT=10ms ./auction-simulator

# Test graceful shutdown (press Ctrl+C during execution)
./auction-simulator
# then press Ctrl+C — only completed auctions will be saved
```
