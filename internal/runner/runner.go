package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/calummacc/g0/internal/httpclient"
)

// Config holds the configuration for a load test
type Config struct {
	URLs        []string // URLs to test (supports multiple endpoints)
	Concurrency int
	Duration    time.Duration
	Method      string
	Body        string
	Headers     map[string]string
	MaxRPS      int // Maximum requests per second (0 = no limit)
}

// RunResult contains both the stats instance (for progress monitoring) and the final summary
type RunResult struct {
	Stats   *Stats
	Summary *Summary
}

// Run executes a load test with the given configuration
func Run(config Config) (*Summary, error) {
	result, err := RunWithStats(config)
	if err != nil {
		return nil, err
	}
	return result.Summary, nil
}

// RunWithStats executes a load test and returns both stats (for progress monitoring) and summary
// statsChan can be used to receive the stats instance immediately after creation (for progress monitoring)
func RunWithStats(config Config) (*RunResult, error) {
	return RunWithStatsAndChannel(config, nil)
}

// RunWithStatsAndChannel executes a load test and optionally sends stats instance to a channel when created
func RunWithStatsAndChannel(config Config, statsChan chan<- *Stats) (*RunResult, error) {
	// Validate URLs
	if len(config.URLs) == 0 {
		return nil, fmt.Errorf("at least one URL is required")
	}

	// Create HTTP client
	client := httpclient.New()

	// Create URL rotator for round-robin distribution
	urlRotator := NewURLRotator(config.URLs)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	// Create results channel
	results := make(chan Result, config.Concurrency*10)

	// Create stats collector
	stats := NewStats()

	// Send stats instance to channel if provided (for progress monitoring)
	if statsChan != nil {
		select {
		case statsChan <- stats:
		default:
			// Channel is full or closed, continue anyway
		}
	}

	// Start stats collector goroutine
	statsDone := make(chan struct{})
	go func() {
		defer close(statsDone)
		for {
			select {
			case result, ok := <-results:
				if !ok {
					return
				}
				stats.AddResult(result)
			case <-ctx.Done():
				// Drain remaining results after context is done
				for {
					select {
					case result := <-results:
						stats.AddResult(result)
					default:
						return
					}
				}
			}
		}
	}()

	// Create rate limiter if MaxRPS is specified
	var rateLimiter *RateLimiter
	if config.MaxRPS > 0 {
		rateLimiter = NewRateLimiter(config.MaxRPS)
		defer rateLimiter.Stop()
	}

	// Use WaitGroup to wait for all workers to finish
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		// Create base request configuration (URL will be selected dynamically)
		baseRequest := httpclient.Request{
			Method:  config.Method,
			Body:    config.Body,
			Headers: config.Headers,
		}
		worker := NewWorker(client, baseRequest, results, rateLimiter, urlRotator)
		go func() {
			defer wg.Done()
			worker.Start(ctx)
		}()
	}

	// Wait for duration to complete
	<-ctx.Done()

	// Wait for all workers to finish (they will stop when ctx.Done() is triggered)
	wg.Wait()

	// Close results channel to signal stats collector to finish
	// This is safe now because all workers have stopped
	close(results)

	// Wait for stats collector to finish processing
	<-statsDone

	// Finalize stats
	stats.Finalize()

	// Get summary
	summary := stats.GetSummary()

	return &RunResult{
		Stats:   stats,
		Summary: &summary,
	}, nil
}

