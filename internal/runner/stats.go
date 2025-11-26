package runner

import (
	"sync"
	"time"
)

// Result represents a single request result
type Result struct {
	Latency    time.Duration
	StatusCode int
	Error      error
}

// Stats aggregates statistics from all requests
type Stats struct {
	mu sync.RWMutex

	TotalRequests    int64
	SuccessRequests  int64
	FailedRequests   int64
	StatusCodeCounts map[int]int64
	Latencies        []time.Duration
	StartTime        time.Time
	EndTime          time.Time
}

// NewStats creates a new Stats instance
func NewStats() *Stats {
	return &Stats{
		StatusCodeCounts: make(map[int]int64),
		Latencies:        make([]time.Duration, 0),
		StartTime:        time.Now(),
	}
}

// AddResult adds a result to the statistics
func (s *Stats) AddResult(result Result) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalRequests++
	s.Latencies = append(s.Latencies, result.Latency)

	if result.Error != nil || result.StatusCode >= 400 {
		s.FailedRequests++
	} else {
		s.SuccessRequests++
	}

	// Record status code, including 0 for network errors
	// StatusCode 0 indicates network/connection errors (not HTTP status codes)
	if result.Error != nil && result.StatusCode == 0 {
		// Network error: use 0 to represent connection/network errors
		s.StatusCodeCounts[0]++
	} else if result.StatusCode > 0 {
		// Valid HTTP status code
		s.StatusCodeCounts[result.StatusCode]++
	}
	// Note: If StatusCode is 0 and Error is nil, it shouldn't happen in normal flow
}

// Finalize marks the end of the test
func (s *Stats) Finalize() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.EndTime = time.Now()
}

// GetSummary returns a summary of the statistics
func (s *Stats) GetSummary() Summary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.Latencies) == 0 {
		return Summary{
			TotalRequests:   s.TotalRequests,
			SuccessRequests: s.SuccessRequests,
			FailedRequests:  s.FailedRequests,
			StatusCodeCounts: s.StatusCodeCounts,
		}
	}

	// Calculate latency statistics
	var min, max, sum time.Duration
	min = s.Latencies[0]
	max = s.Latencies[0]

	for _, lat := range s.Latencies {
		if lat < min {
			min = lat
		}
		if lat > max {
			max = lat
		}
		sum += lat
	}

	avg := sum / time.Duration(len(s.Latencies))

	// Calculate percentiles
	p90 := Percentile(s.Latencies, 90)
	p95 := Percentile(s.Latencies, 95)
	p99 := Percentile(s.Latencies, 99)

	// Calculate RPS
	duration := s.EndTime.Sub(s.StartTime)
	var rps float64
	if duration > 0 {
		rps = float64(s.TotalRequests) / duration.Seconds()
	}

	return Summary{
		TotalRequests:    s.TotalRequests,
		SuccessRequests:  s.SuccessRequests,
		FailedRequests:   s.FailedRequests,
		StatusCodeCounts: s.StatusCodeCounts,
		MinLatency:       min,
		MaxLatency:       max,
		AvgLatency:       avg,
		P90Latency:       p90,
		P95Latency:       p95,
		P99Latency:       p99,
		RPS:              rps,
		Duration:         duration,
	}
}

// ProgressStats contains current progress statistics (for real-time display)
type ProgressStats struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
}

// GetProgressStats returns current progress statistics without locking for long operations
func (s *Stats) GetProgressStats() ProgressStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return ProgressStats{
		TotalRequests:   s.TotalRequests,
		SuccessRequests: s.SuccessRequests,
		FailedRequests:  s.FailedRequests,
	}
}

// Summary contains aggregated statistics
type Summary struct {
	TotalRequests    int64
	SuccessRequests  int64
	FailedRequests   int64
	StatusCodeCounts map[int]int64
	MinLatency       time.Duration
	MaxLatency       time.Duration
	AvgLatency       time.Duration
	P90Latency       time.Duration
	P95Latency       time.Duration
	P99Latency       time.Duration
	RPS              float64
	Duration         time.Duration
}

