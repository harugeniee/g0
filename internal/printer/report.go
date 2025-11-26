package printer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calummacc/g0/internal/runner"
)

// PrintLogo prints the g0 logo
func PrintLogo() {
	logo := `
	┌───────────────────────────────┐
	│             g0                │
	│    Mini High-Speed Load Tool  │
	└───────────────────────────────┘
  	`

	fmt.Print(logo)
	fmt.Println()
}

// PrintTestStart prints the test configuration
func PrintTestStart(urls []string, concurrency int, duration time.Duration) {
	fmt.Println("Load Test Started")
	if len(urls) == 1 {
		fmt.Printf("URL: %s\n", urls[0])
	} else {
		fmt.Printf("URLs (%d endpoints):\n", len(urls))
		for i, url := range urls {
			fmt.Printf("  %d. %s\n", i+1, url)
		}
	}
	fmt.Printf("Concurrency: %d\n", concurrency)
	fmt.Printf("Duration: %s\n", duration)
	fmt.Println()
}

// PrintResults prints the test results in a formatted way
func PrintResults(summary *runner.Summary) {
	fmt.Println("Results:")
	fmt.Printf("Total Requests: %d\n", summary.TotalRequests)
	fmt.Printf("Success: %d\n", summary.SuccessRequests)
	fmt.Printf("Failed: %d\n", summary.FailedRequests)
	fmt.Printf("RPS: %.1f\n", summary.RPS)
	fmt.Println()

	fmt.Println("Latency:")
	fmt.Printf("  Min: %s\n", formatDuration(summary.MinLatency))
	fmt.Printf("  Avg: %s\n", formatDuration(summary.AvgLatency))
	fmt.Printf("  Max: %s\n", formatDuration(summary.MaxLatency))
	fmt.Printf("  p90: %s\n", formatDuration(summary.P90Latency))
	fmt.Printf("  p95: %s\n", formatDuration(summary.P95Latency))
	fmt.Printf("  p99: %s\n", formatDuration(summary.P99Latency))

	// Print status code distribution if there are any
	if len(summary.StatusCodeCounts) > 0 {
		fmt.Println()
		fmt.Println("Status Codes:")
		for code, count := range summary.StatusCodeCounts {
			fmt.Printf("  %d: %d\n", code, count)
		}
	}
}

// PrintProgress displays a progress bar with current test statistics
// It updates in-place on the same line using carriage return
// spinnerFrame is used for animation when generating report (0-3 for spinner animation)
func PrintProgress(elapsed time.Duration, totalDuration time.Duration, stats *runner.ProgressStats, spinnerFrame int) {
	// Calculate progress percentage
	progress := float64(elapsed) / float64(totalDuration)
	isComplete := progress >= 1.0
	if progress > 1.0 {
		progress = 1.0
	}

	// Create progress bar (40 characters wide for better display)
	barWidth := 40
	filled := int(progress * float64(barWidth))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Calculate current RPS
	var rps float64
	if elapsed > 0 {
		rps = float64(stats.TotalRequests) / elapsed.Seconds()
	}

	// Format elapsed time
	elapsedStr := formatDurationShort(elapsed)
	totalStr := formatDurationShort(totalDuration)

	// Spinner characters for animation
	spinnerChars := []string{"|", "/", "-", "\\"}

	// ANSI escape code to clear the line: \033[2K clears entire line, \r returns to start
	clearLine := "\033[2K\r"

	// If test is complete, show "Generating report..." message with spinner
	if isComplete {
		spinner := spinnerChars[spinnerFrame%len(spinnerChars)]
		fmt.Fprintf(os.Stderr, "%s[%s] 100.0%% | Generating report %s | Req: %d | ✓: %d | ✗: %d | RPS: %.1f   ",
			clearLine, strings.Repeat("█", barWidth), spinner, stats.TotalRequests, stats.SuccessRequests, stats.FailedRequests, rps)
	} else {
		// Print progress on the same line (using clearLine to clear and return to start)
		// Add spaces at the end to clear any remaining characters from previous updates
		fmt.Fprintf(os.Stderr, "%s[%s] %.1f%% | %s/%s | Req: %d | ✓: %d | ✗: %d | RPS: %.1f   ",
			clearLine, bar, progress*100, elapsedStr, totalStr,
			stats.TotalRequests, stats.SuccessRequests, stats.FailedRequests, rps)
	}

	// Flush to ensure immediate display
	os.Stderr.Sync()
}

// PrintGeneratingReport displays a one-time "Generating report..." message
func PrintGeneratingReport(stats *runner.ProgressStats, rps float64) {
	barWidth := 40
	bar := strings.Repeat("█", barWidth)
	// Clear line and print final message
	fmt.Fprintf(os.Stderr, "\033[2K\r[%s] 100.0%% | Generating report... | Req: %d | ✓: %d | ✗: %d | RPS: %.1f   ",
		bar, stats.TotalRequests, stats.SuccessRequests, stats.FailedRequests, rps)
	os.Stderr.Sync()
}

// ClearProgress clears the progress line
func ClearProgress() {
	// Clear the entire line by printing spaces and returning to start
	fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 200))
	os.Stderr.Sync()
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%.2fns", float64(d.Nanoseconds()))
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.2fµs", float64(d.Nanoseconds())/1000.0)
	} else if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1000000.0)
	}
	return d.Round(time.Millisecond).String()
}

// formatDurationShort formats a duration in a short, readable way for progress display
func formatDurationShort(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Nanoseconds())/1000000.0)
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

// JSONOutput represents the JSON structure for test results
type JSONOutput struct {
	Metadata JSONMetadata `json:"metadata"`
	Metrics  JSONMetrics  `json:"metrics"`
}

// JSONMetadata contains test configuration and timing information
type JSONMetadata struct {
	URL         string            `json:"url,omitempty"`         // Single URL (if only one)
	URLs        []string          `json:"urls,omitempty"`        // Multiple URLs (if more than one)
	Method      string            `json:"method"`
	Concurrency int               `json:"concurrency"`
	Duration    string            `json:"duration"`
	DurationMs  int64             `json:"duration_ms"`
	Headers     map[string]string `json:"headers,omitempty"`
	StartTime   string            `json:"start_time,omitempty"`
	EndTime     string            `json:"end_time,omitempty"`
}

// JSONMetrics contains all test metrics
type JSONMetrics struct {
	Requests    JSONRequests     `json:"requests"`
	Latency     JSONLatency      `json:"latency"`
	StatusCodes map[string]int64 `json:"status_codes"`
}

// JSONRequests contains request statistics
type JSONRequests struct {
	Total   int64   `json:"total"`
	Success int64   `json:"success"`
	Failed  int64   `json:"failed"`
	RPS     float64 `json:"rps"`
}

// JSONLatency contains latency statistics
type JSONLatency struct {
	Min JSONDuration `json:"min"`
	Max JSONDuration `json:"max"`
	Avg JSONDuration `json:"avg"`
	P90 JSONDuration `json:"p90"`
	P95 JSONDuration `json:"p95"`
	P99 JSONDuration `json:"p99"`
}

// JSONDuration represents a duration in both human-readable and numeric formats
type JSONDuration struct {
	Value string  `json:"value"` // Human-readable format (e.g., "12.45ms")
	Ms    float64 `json:"ms"`    // Duration in milliseconds
}

// PrintResultsJSON prints the test results in JSON format and saves to file
// Returns the file path where JSON was saved
func PrintResultsJSON(summary *runner.Summary, urls []string, concurrency int, duration time.Duration, method string, headers map[string]string, outputFile string) (string, error) {
	// Convert status codes map from int keys to string keys for JSON
	// Status code 0 represents network/connection errors
	statusCodes := make(map[string]int64)
	for code, count := range summary.StatusCodeCounts {
		if code == 0 {
			// Use "error" or "0" for network errors to make it clearer
			statusCodes["error"] = count
		} else {
			statusCodes[fmt.Sprintf("%d", code)] = count
		}
	}

	// Build JSON output structure
	metadata := JSONMetadata{
		Method:      method,
		Concurrency: concurrency,
		Duration:    duration.String(),
		DurationMs:  duration.Milliseconds(),
		Headers:     headers,
	}
	
	// Set URL or URLs based on count
	if len(urls) == 1 {
		metadata.URL = urls[0]
	} else {
		metadata.URLs = urls
	}
	
	output := JSONOutput{
		Metadata: metadata,
		Metrics: JSONMetrics{
			Requests: JSONRequests{
				Total:   summary.TotalRequests,
				Success: summary.SuccessRequests,
				Failed:  summary.FailedRequests,
				RPS:     summary.RPS,
			},
			Latency: JSONLatency{
				Min: durationToJSON(summary.MinLatency),
				Max: durationToJSON(summary.MaxLatency),
				Avg: durationToJSON(summary.AvgLatency),
				P90: durationToJSON(summary.P90Latency),
				P95: durationToJSON(summary.P95Latency),
				P99: durationToJSON(summary.P99Latency),
			},
			StatusCodes: statusCodes,
		},
	}

	// Marshal to JSON with indentation for readability
	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Determine output file path
	var filePath string
	if outputFile != "" {
		// Use user-specified file path
		filePath = outputFile
		// Create directory if it doesn't exist
		dir := filepath.Dir(filePath)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", fmt.Errorf("failed to create output directory: %w", err)
			}
		}
	} else {
		// Generate default file path in results/ directory
		resultsDir := "results"
		if err := os.MkdirAll(resultsDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create results directory: %w", err)
		}

		// Generate filename with timestamp: g0-result-YYYYMMDD-HHMMSS.json
		timestamp := time.Now().Format("20060102-150405")
		filePath = filepath.Join(resultsDir, fmt.Sprintf("g0-result-%s.json", timestamp))
	}

	// Write JSON to file
	if err := os.WriteFile(filePath, jsonBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write JSON file: %w", err)
	}

	// Don't print JSON to stdout - results are already shown in text format
	// JSON is only saved to file

	return filePath, nil
}

// durationToJSON converts a time.Duration to JSONDuration format
func durationToJSON(d time.Duration) JSONDuration {
	return JSONDuration{
		Value: formatDuration(d),
		Ms:    float64(d.Nanoseconds()) / 1000000.0, // Convert to milliseconds
	}
}
