package runner

import (
	"sync/atomic"
)

// URLRotator provides round-robin URL rotation for load testing multiple endpoints
type URLRotator struct {
	urls []string
	idx  int64 // Atomic counter for round-robin selection
}

// NewURLRotator creates a new URL rotator with the given URLs
func NewURLRotator(urls []string) *URLRotator {
	if len(urls) == 0 {
		return nil
	}
	return &URLRotator{
		urls: urls,
		idx:  0,
	}
}

// Next returns the next URL in round-robin fashion
// Thread-safe using atomic operations
func (r *URLRotator) Next() string {
	if r == nil || len(r.urls) == 0 {
		return ""
	}
	if len(r.urls) == 1 {
		return r.urls[0]
	}
	
	// Atomic increment and modulo for thread-safe round-robin
	idx := atomic.AddInt64(&r.idx, 1) - 1
	return r.urls[int(idx)%len(r.urls)]
}

