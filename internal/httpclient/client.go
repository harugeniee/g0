package httpclient

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
)

// Client wraps http.Client with keep-alive enabled
type Client struct {
	httpClient *http.Client
}

// New creates a new HTTP client with keep-alive enabled
func New() *Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}
}

// Request represents an HTTP request configuration
type Request struct {
	Method  string
	URL     string
	Body    string
	Headers map[string]string
	Context context.Context // Context for request cancellation
}

// Response represents the result of an HTTP request
type Response struct {
	StatusCode int
	Latency    time.Duration
	Error      error
}

// Do performs an HTTP request and returns the response
func (c *Client) Do(req Request) Response {
	start := time.Now()

	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = bytes.NewBufferString(req.Body)
	}

	// Use context-aware request creation to support cancellation
	// If no context is provided, use context.Background()
	ctx := req.Context
	if ctx == nil {
		ctx = context.Background()
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bodyReader)
	if err != nil {
		return Response{
			StatusCode: 0,
			Latency:    time.Since(start),
			Error:      err,
		}
	}

	// Set headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Perform the request
	resp, err := c.httpClient.Do(httpReq)
	latency := time.Since(start)

	if err != nil {
		return Response{
			StatusCode: 0,
			Latency:    latency,
			Error:      err,
		}
	}
	defer resp.Body.Close()

	return Response{
		StatusCode: resp.StatusCode,
		Latency:    latency,
		Error:      nil,
	}
}

