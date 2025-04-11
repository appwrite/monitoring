# Appwrite System Monitoring

A system monitoring tool for Appwrite servers that tracks CPU, memory, and disk usage with alerting via BetterStack.

## Features

- Monitors CPU usage with configurable thresholds
- Monitors memory usage with configurable thresholds
- Monitors disk usage (root and mounted volumes) with configurable thresholds
- Uses Exponential Moving Average (EMA) to smooth out short-term spikes
- Sends alerts to BetterStack when thresholds are exceeded

## Project Structure

The project follows a simplified package structure:

```
monitoring/
├── main.go            # Entry point with CLI argument parsing
├── pkg/
│   ├── logger.go      # Logging functionality at package level
│   └── monitor/       # Core monitoring functionality
│       ├── monitor.go # Main monitoring struct and Metric model
│       ├── cpu.go     # CPU-specific monitoring
│       ├── memory.go  # Memory-specific monitoring
│       └── disk.go    # Disk-specific monitoring
├── go.mod             # Go module definition
└── README.md          # Documentation
```

## Usage

### Building

```bash
# Clone the repository
git clone https://github.com/appwrite/monitoring.git
cd monitoring

# Build the binary
go build -o appwrite-monitor main.go
```

### Running

```bash
# Basic usage
./appwrite-monitor --url="https://betterstack-webhook-url"

# With custom thresholds
./appwrite-monitor \
  --url="https://betterstack-webhook-url" \
  --interval=60 \
  --cpu-limit=80 \
  --memory-limit=85 \
  --disk-limit=90
```

### Command Line Options

- `--url`: BetterStack webhook URL (required)
- `--interval`: Check interval in seconds (default: 300)
- `--cpu-limit`: CPU usage threshold percentage (default: 90)
- `--memory-limit`: Memory usage threshold percentage (default: 90)
- `--disk-limit`: Disk usage threshold percentage (default: 85)

## How It Works

The monitoring tool uses Exponential Moving Average (EMA) to track resource usage over time, which helps prevent false alerts from momentary spikes. When the EMA of a resource exceeds the configured threshold, an alert is sent to BetterStack.

The EMA smoothing factor is automatically calculated based on the check interval to provide roughly 5 minutes of smoothing. This means that sudden spikes will have less impact on the reported values, while sustained high usage will still trigger alerts.
