package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/appwrite/monitoring/pkg"
	"github.com/appwrite/monitoring/pkg/monitor"
)

func main() {
	log := pkg.NewLogger()

	// Command line flags
	betterStackURL := flag.String("url", "", "BetterStack webhook URL (required)")
	interval := flag.Int("interval", 300, "Check interval in seconds (default: 300)")
	cpuLimit := flag.Float64("cpu-limit", 90.0, "CPU usage threshold percentage (default: 90)")
	memoryLimit := flag.Float64("memory-limit", 90.0, "Memory usage threshold percentage (default: 90)")
	diskLimit := flag.Float64("disk-limit", 85.0, "Disk usage threshold percentage (default: 85)")

	// Add usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	// Validate required flags
	if *betterStackURL == "" {
		flag.Usage()
		log.Fatal("BetterStack webhook URL is required")
	}

	// Validate ranges
	if *interval <= 0 {
		log.Fatal("Interval must be greater than 0")
	}
	if *cpuLimit < 0 || *cpuLimit > 100 {
		log.Fatal("CPU limit must be between 0 and 100")
	}
	if *memoryLimit < 0 || *memoryLimit > 100 {
		log.Fatal("Memory limit must be between 0 and 100")
	}
	if *diskLimit < 0 || *diskLimit > 100 {
		log.Fatal("Disk limit must be between 0 and 100")
	}

	monitor, err := monitor.NewSystemMonitor(*betterStackURL, *interval, *cpuLimit, *memoryLimit, *diskLimit)
	if err != nil {
		log.Fatal("Failed to create system monitor: %v", err)
	}

	log.Info("Starting monitoring with settings:")
	log.Info("- Check interval: %d seconds", *interval)
	log.Info("- CPU limit: %.1f%%", *cpuLimit)
	log.Info("- Memory limit: %.1f%%", *memoryLimit)
	log.Info("- Disk limit: %.1f%%", *diskLimit)

	monitor.Start()
} 