# System Monitoring

A lightweight system monitoring tool that tracks CPU, memory, and disk usage across your infrastructure. When resource usage exceeds defined thresholds, it creates incidents in BetterStack (formerly BetterUptime).

## Features

- CPU usage monitoring
- Memory usage monitoring
- Disk usage monitoring (root and mounted volumes)
- Automatic incident creation and resolution
- Configurable thresholds via CLI
- Docker-based deployment

## Command Line Usage

The monitoring tool is configured through command-line flags:

```bash
monitoring [flags]

Flags:
  -url string
        BetterStack webhook URL (required)
  -interval int
        Check interval in seconds (default: 300)
  -cpu-limit float
        CPU usage threshold percentage (default: 90)
  -memory-limit float
        Memory usage threshold percentage (default: 90)
  -disk-limit float
        Disk usage threshold percentage (default: 85)
  -help
        Display help information
```

### Examples

```bash
# Basic usage with required URL
monitoring --url=https://betterstack.com/webhook/xyz

# Custom thresholds
monitoring --url=https://betterstack.com/webhook/xyz \
          --cpu-limit=95 \
          --memory-limit=85 \
          --disk-limit=80

# More frequent checks (every minute)
monitoring --url=https://betterstack.com/webhook/xyz --interval=60
```

## Docker Deployment

### Using Docker Run

```bash
docker run -d \
  --name monitoring \
  --privileged \
  --pid=host \
  -v /:/host:ro \
  ghcr.io/appwrite/monitoring:latest \
  monitoring \
  --url=https://betterstack.com/webhook/xyz \
  --interval=300 \
  --cpu-limit=90 \
  --memory-limit=90 \
  --disk-limit=85
```

### Using Docker Compose

The docker-compose.yml file is configured with default parameters that you can modify as needed:

```bash
docker-compose up -d
```

To modify the parameters, edit the command section in docker-compose.yml:
```yaml
command:
  - monitoring
  - "--url=https://betterstack.com/webhook/xyz"
  - "--interval=10"
  - "--cpu-limit=90"
  - "--memory-limit=80"
  - "--disk-limit=85"
```

## Building from Source

1. Clone the repository:
```bash
git clone https://github.com/appwrite/monitoring.git
cd monitoring
```

2. Build the binary:
```bash
go build -o monitoring
```

3. Run the monitoring tool:
```bash
monitoring --url=https://betterstack.com/webhook/xyz
```

## Development

### Requirements
- Go 1.21 or later
- Docker and Docker Compose (for containerized deployment)

### Local Development
1. Install dependencies:
```bash
go mod download
```

2. Build and run:
```bash
go build -o monitoring
monitoring --url=https://betterstack.com/webhook/xyz
```

### Docker Development
1. Build the image:
```bash
docker build -t monitoring .
```

2. Run with Docker:
```bash
# Show help
docker run --rm ghcr.io/appwrite/monitoring:latest monitoring --help

# Run monitoring with custom parameters
docker run -d \
  --name monitoring \
  --privileged \
  --pid=host \
  -v /:/host:ro \
  ghcr.io/appwrite/monitoring:latest \
  monitoring \
  --url=https://betterstack.com/webhook/xyz \
  --interval=60
```

## License

MIT License - see the [LICENSE](LICENSE) file for details
