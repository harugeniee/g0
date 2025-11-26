package runner

import (
	"context"

	"github.com/calummacc/g0/internal/httpclient"
)

// Worker sends HTTP requests in a loop until the context is cancelled
type Worker struct {
	client      *httpclient.Client
	request     httpclient.Request // Base request config (URL will be selected dynamically)
	results     chan<- Result
	rateLimiter *RateLimiter
	urlRotator  *URLRotator // For selecting URL in round-robin fashion
}

// NewWorker creates a new worker
func NewWorker(client *httpclient.Client, request httpclient.Request, results chan<- Result, rateLimiter *RateLimiter, urlRotator *URLRotator) *Worker {
	return &Worker{
		client:      client,
		request:     request,
		results:     results,
		rateLimiter: rateLimiter,
		urlRotator:  urlRotator,
	}
}

// Start begins the worker loop, sending requests until ctx is cancelled
func (w *Worker) Start(ctx context.Context) {
	defer func() {
		// Recover from any panic (e.g., sending on closed channel)
		// This should not happen with proper synchronization, but provides safety
		recover()
	}()

	for {
		// Check if context is done before starting a new request
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Wait for rate limiter token if rate limiting is enabled
		if !w.rateLimiter.Wait(ctx) {
			// Context cancelled or rate limiter stopped
			return
		}

		// Select URL from rotator (round-robin)
		selectedURL := w.urlRotator.Next()
		if selectedURL == "" {
			// No URL available, skip
			continue
		}

		// Create request with selected URL and context for cancellation
		request := w.request
		request.URL = selectedURL
		request.Context = ctx // Pass context to enable request cancellation

		// Send request
		resp := w.client.Do(request)

		// Check context again before sending result (request might have taken time)
		select {
		case <-ctx.Done():
			// Context cancelled, don't send result
			return
		case w.results <- Result{
			Latency:    resp.Latency,
			StatusCode: resp.StatusCode,
			Error:      resp.Error,
		}:
			// Successfully sent result, continue loop
		}
	}
}

