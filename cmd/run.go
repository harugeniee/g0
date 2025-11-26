package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/calummacc/g0/internal/printer"
	"github.com/calummacc/g0/internal/runner"
	"github.com/spf13/cobra"
)

var (
	urls        []string
	concurrency int
	duration    string
	method      string
	body        string
	headers     []string
	jsonOutput  bool
	outputFile  string
	maxRPS      int
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a load test",
	Long: `Run a load test against a target URL with specified concurrency and duration.

Example:
  g0 run --url https://api.example.com --c 100 --d 10s
  g0 run --url https://api.example.com --c 50 --d 30s --method POST --body '{"key":"value"}' --headers "Content-Type: application/json"`,
	RunE: runLoadTest,
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringArrayVarP(&urls, "url", "u", []string{}, "Target URL(s) - can be specified multiple times (required)")
	runCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 10, "Number of concurrent workers")
	runCmd.Flags().StringVarP(&duration, "duration", "d", "10s", "Test duration (e.g., 10s, 1m, 30s)")
	runCmd.Flags().StringVarP(&method, "method", "m", "GET", "HTTP method")
	runCmd.Flags().StringVarP(&body, "body", "b", "", "Request body")
	runCmd.Flags().StringArrayVarP(&headers, "headers", "H", []string{}, "HTTP headers (can be specified multiple times)")
	runCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output results in JSON format")
	runCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path for JSON results (default: results/g0-result-YYYYMMDD-HHMMSS.json)")
	runCmd.Flags().IntVarP(&maxRPS, "max-rps", "r", 0, "Maximum requests per second (0 = no limit)")

	runCmd.MarkFlagRequired("url")
}

func runLoadTest(cmd *cobra.Command, args []string) error {
	// Parse duration
	testDuration, err := time.ParseDuration(duration)
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}

	// Validate URLs
	if len(urls) == 0 {
		return fmt.Errorf("at least one URL is required (use --url or -u)")
	}

	// Validate concurrency
	if concurrency <= 0 {
		return fmt.Errorf("concurrency must be greater than 0")
	}

	// Parse headers
	headerMap := make(map[string]string)
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid header format: %s (expected 'Key: Value')", h)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		headerMap[key] = value
	}

	// Print logo
	printer.PrintLogo()

	// Print test configuration
	printer.PrintTestStart(urls, concurrency, testDuration)

	// Validate max RPS if specified
	if maxRPS < 0 {
		return fmt.Errorf("max-rps must be greater than or equal to 0")
	}

	// Create and run the load test
	config := runner.Config{
		URLs:        urls,
		Concurrency: concurrency,
		Duration:    testDuration,
		Method:      method,
		Body:        body,
		Headers:     headerMap,
		MaxRPS:      maxRPS,
	}

	// Channel to receive test result
	resultChan := make(chan *runner.RunResult, 1)
	errChan := make(chan error, 1)
	statsChan := make(chan *runner.Stats, 1)

	// Start progress monitoring in a goroutine
	progressDone := make(chan struct{})
	testCompleted := make(chan struct{}) // Signal when test is actually done
	startTime := time.Now()
	var stats *runner.Stats

	// Start the test in a goroutine
	go func() {
		result, err := runner.RunWithStatsAndChannel(config, statsChan)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- result
	}()

	// Progress monitoring goroutine
	go func() {
		// Wait for stats to be available
		select {
		case s := <-statsChan:
			stats = s
		case <-time.After(2 * time.Second):
			// Stats not available yet, continue anyway (shouldn't happen normally)
		}

		ticker := time.NewTicker(100 * time.Millisecond) // Update every 100ms
		defer ticker.Stop()

		for {
			select {
			case s := <-statsChan:
				// Stats instance is now available (if not received earlier)
				stats = s
			case <-ticker.C:
				// Check if test completed first - if so, stop immediately
				select {
				case <-testCompleted:
					return
				case <-progressDone:
					return
				default:
					// Test still running, continue updating
					elapsed := time.Since(startTime)
					// Only update if elapsed < testDuration (don't show 100% from progress goroutine)
					// Main goroutine will handle 100% and "Generating report" display
					if elapsed < testDuration {
						if stats != nil {
							progressStats := stats.GetProgressStats()
							printer.PrintProgress(elapsed, testDuration, &progressStats, 0)
						} else {
							// Stats not available yet, show basic progress with zero stats
							zeroStats := runner.ProgressStats{}
							printer.PrintProgress(elapsed, testDuration, &zeroStats, 0)
						}
					}
					// If elapsed >= testDuration, don't update anymore - let main goroutine handle it
				}
			case <-progressDone:
				// Stop immediately when test is done
				return
			case <-testCompleted:
				// Test completed, stop updating
				return
			}
		}
	}()

	// Wait for test to complete
	var result *runner.RunResult
	select {
	case err := <-errChan:
		close(progressDone)
		time.Sleep(50 * time.Millisecond)
		printer.ClearProgress()
		return fmt.Errorf("load test failed: %w", err)
	case result = <-resultChan:
		// Test completed - signal to stop progress updates immediately
		// Close testCompleted first to signal completion
		close(testCompleted)
		// Then close progressDone to ensure goroutine stops
		close(progressDone)
		// Wait longer to ensure all ticker events are processed and goroutine has stopped
		time.Sleep(250 * time.Millisecond)
		
		// Show final "Generating report..." message once
		if stats != nil {
			progressStats := stats.GetProgressStats()
			var rps float64
			if testDuration > 0 {
				rps = float64(progressStats.TotalRequests) / testDuration.Seconds()
			}
			printer.PrintGeneratingReport(&progressStats, rps)
			time.Sleep(300 * time.Millisecond) // Show message briefly
		}
		
		// Clear progress line
		printer.ClearProgress()
		fmt.Println() // Add a newline after clearing progress
	}

	// Print results in text format
	printer.PrintResults(result.Summary)
	
	// If JSON output is enabled, also save to file
	if jsonOutput {
		filePath, err := printer.PrintResultsJSON(result.Summary, urls, concurrency, testDuration, method, headerMap, outputFile)
		if err != nil {
			return fmt.Errorf("failed to save JSON output: %w", err)
		}
		fmt.Fprintf(os.Stderr, "\nResults saved to: %s\n", filePath)
	}

	return nil
}
