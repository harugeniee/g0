# g0

A minimal, high-performance HTTP load tester written in Go. Inspired by k6, but designed to be lightweight and fast.

## Features

- **High Performance**: Built with Go's concurrency primitives for maximum throughput
- **Simple CLI**: Easy-to-use command-line interface
- **Rich Metrics**: Comprehensive statistics including latency percentiles (p90, p95, p99)
- **Keep-Alive**: HTTP connection pooling for efficient request handling
- **Duration-Based**: Run tests for a specified duration
- **Flexible**: Support for custom methods, headers, and request bodies

## Installation

### Build from Source

#### Prerequisites

- Go 1.21 or later
- Make (optional, for using Makefile)

#### Quick Build

```bash
# Clone the repository
git clone https://github.com/calummacc/g0.git
cd g0

# Build for your current platform
make build

# Or build directly with Go
go build -o g0 main.go
```

#### Cross-Platform Builds

The Makefile provides convenient commands to build for different platforms:

**macOS:**
```bash
make build-macos    # Build macOS binary
make pkg-macos      # Create macOS installer package (.pkg)
make dmg            # Create macOS disk image (.dmg)
```

**Windows:**
```bash
make build-windows  # Build Windows binary (.exe)
make pkg-windows    # Create Windows package (.zip)
```

**Linux:**
```bash
make build-linux    # Build Linux binary
make pkg-linux      # Create Linux package (.tar.gz)
```

**All Platforms:**
```bash
# Build for all platforms
make build-macos build-windows build-linux

# Create packages for all platforms
make pkg-macos pkg-windows pkg-linux
```

#### Available Make Targets

| Command | Description |
|---------|-------------|
| `make build` | Build for current platform |
| `make build-macos` | Build macOS binary |
| `make build-windows` | Build Windows binary (.exe) |
| `make build-linux` | Build Linux binary |
| `make pkg-macos` | Create macOS installer (.pkg) |
| `make pkg-windows` | Create Windows package (.zip) |
| `make pkg-linux` | Create Linux package (.tar.gz) |
| `make dmg` | Create macOS disk image (.dmg) |
| `make install-pkg` | Install macOS package (requires sudo) |
| `make clean` | Remove build artifacts |
| `make clean-pkg` | Remove package artifacts |
| `make test` | Run test suite |
| `make help` | Show all available targets |

### Using Go Install

```bash
go install github.com/calummacc/g0@latest
```

## Usage

### Basic Example

```bash
g0 run --url https://api.example.com --c 100 --d 10s
```

### Command Options

```
Flags:
  -u, --url stringArray  Target URL(s) - can be specified multiple times (required)
  -c, --concurrency int   Number of concurrent workers (default 10)
  -d, --duration string   Test duration (e.g., 10s, 1m, 30s) (default "10s")
  -m, --method string     HTTP method (default "GET")
  -b, --body string       Request body
  -H, --headers strings   HTTP headers (can be specified multiple times)
  -j, --json              Output results in JSON format
  -o, --output string     Output file path for JSON results (default: results/g0-result-YYYYMMDD-HHMMSS.json)
  -r, --max-rps int      Maximum requests per second (0 = no limit)
```

### Examples

**Simple GET request:**
```bash
g0 run --url https://api.example.com --c 50 --d 30s
```

**POST request with JSON body:**
```bash
g0 run --url https://api.example.com/api/users \
  --method POST \
  --body '{"name":"John","email":"john@example.com"}' \
  --headers "Content-Type: application/json" \
  --c 100 \
  --d 10s
```

**Multiple headers:**
```bash
g0 run --url https://api.example.com \
  --headers "Authorization: Bearer token123" \
  --headers "X-Custom-Header: value" \
  --c 200 \
  --d 1m
```

**JSON output format:**
```bash
# JSON output (automatically saved to results/ directory)
g0 run --url https://api.example.com --c 50 --d 10s --json

# JSON output with custom file path
g0 run --url https://api.example.com --c 50 --d 10s --json --output my-results.json

# JSON output to specific directory
g0 run --url https://api.example.com --c 50 --d 10s --json --output reports/test-result.json
```

**Rate limiting (max RPS):**
```bash
# Limit to 100 requests per second
g0 run --url https://api.example.com --c 50 --d 10s --max-rps 100

# No rate limiting (default, workers send requests as fast as possible)
g0 run --url https://api.example.com --c 50 --d 10s
```

**Multiple URLs/endpoints:**
```bash
# Test multiple endpoints with round-robin distribution
g0 run --url https://api.example.com/v1/users --url https://api.example.com/v1/posts --url https://api.example.com/v1/comments -c 50 -d 10s

# Test different endpoints with different status codes
g0 run --url https://httpbin.org/get --url https://httpbin.org/status/200 --url https://httpbin.org/status/404 -c 20 -d 5s

# Multiple URLs with rate limiting
g0 run --url https://api.example.com/endpoint1 --url https://api.example.com/endpoint2 -c 50 -d 10s --max-rps 100
```

When multiple URLs are specified, requests are distributed in round-robin fashion across all endpoints. This allows you to test load balancing, different API endpoints, or compare performance across multiple services.

When using `--json`, the results are automatically saved to a file in the `results/` directory with a timestamp-based filename (e.g., `results/g0-result-20240101-120000.json`). You can also specify a custom output path using the `--output` flag. The JSON output includes all metrics in a structured format, making it easy to parse and integrate with other tools or scripts. Example output:

```json
{
  "metadata": {
    "url": "https://api.example.com",
    "method": "GET",
    "concurrency": 50,
    "duration": "10s",
    "duration_ms": 10000,
    "headers": {}
  },
  "metrics": {
    "requests": {
      "total": 12004,
      "success": 11800,
      "failed": 204,
      "rps": 1200.4
    },
    "latency": {
      "min": {
        "value": "5.23ms",
        "ms": 5.23
      },
      "max": {
        "value": "85.12ms",
        "ms": 85.12
      },
      "avg": {
        "value": "12.45ms",
        "ms": 12.45
      },
      "p90": {
        "value": "20.34ms",
        "ms": 20.34
      },
      "p95": {
        "value": "24.56ms",
        "ms": 24.56
      },
      "p99": {
        "value": "40.78ms",
        "ms": 40.78
      }
    },
    "status_codes": {
      "200": 11800,
      "500": 204
    }
  }
}
```

## Output Format

```
Load Test Started
URL: https://api.example.com
Concurrency: 100
Duration: 10s

Results:
Total Requests: 12004
Success: 11800
Failed: 204
RPS: 1200.4

Latency:
  Min: 5.23ms
  Avg: 12.45ms
  Max: 85.12ms
  p90: 20.34ms
  p95: 24.56ms
  p99: 40.78ms

Status Codes:
  200: 11800
  500: 204
```

## Architecture

The project follows a clean, modular architecture:

```
g0/
  cmd/
    root.go          # Cobra root command
    run.go           # Run command implementation
  internal/
    runner/
      runner.go      # Main orchestration logic
      worker.go      # Worker goroutines
      stats.go       # Statistics collection
      percentiles.go # Percentile calculations
    httpclient/
      client.go      # HTTP client with keep-alive
    printer/
      report.go      # Output formatting
  main.go            # Entry point
  go.mod
```

## How It Works

1. **Workers**: Spawns N concurrent worker goroutines (specified by `--concurrency`)
2. **Request Loop**: Each worker continuously sends HTTP requests until the duration expires
3. **Results Channel**: Results are sent through a channel to a stats collector
4. **Statistics**: Aggregates metrics including:
   - Total requests, success/failure counts
   - Status code distribution
   - Latency statistics (min, max, avg, percentiles)
   - Requests per second (RPS)
5. **Output**: Displays formatted results to the console

## Performance Considerations

- Uses HTTP keep-alive connections for efficient request handling
- Connection pooling with configurable limits
- Lock-free statistics collection where possible
- Efficient percentile calculation using sorting and interpolation

## Future Improvements (v2/v3)

### v2 Features
- [x] Real-time progress updates during test execution
- [x] JSON output format option
- [x] Request rate limiting (e.g., max RPS)
- [x] Support for multiple URLs/endpoints
- [ ] Request timeout configuration
- [ ] TLS/SSL configuration options
- [ ] Basic authentication support

### v3 Features
- [ ] Script-based testing (like k6)
- [ ] Response validation and assertions
- [ ] Graph/chart visualization
- [ ] Export results to CSV/JSON
- [ ] Distributed load testing
- [ ] Custom metrics and tags
- [ ] Integration with monitoring systems

## Automated Releases

This project uses GitHub Actions to automatically build and create releases when you push a version tag.

### How to Create a Release

1. **Create and push a version tag:**
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **GitHub Actions will automatically:**
   - Build binaries for macOS, Windows, and Linux
   - Create packages (.pkg, .zip, .tar.gz)
   - Create a GitHub Release with all artifacts attached
   - Generate release notes

3. **Manual trigger (optional):**
   - Go to Actions tab in GitHub
   - Select "Build and Release" workflow
   - Click "Run workflow" to manually trigger

### Release Workflow

The workflow (`.github/workflows/release.yml`) will:
- Build for all platforms in parallel
- Create platform-specific packages
- Upload artifacts
- Create a GitHub Release with download links

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

See [LICENSE](LICENSE) file for details.

## Acknowledgments

Inspired by [k6](https://k6.io/) and other load testing tools.
