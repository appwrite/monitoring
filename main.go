package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type Incident struct {
	Title     string `json:"title"`
	Cause     string `json:"cause"`
	AlertID   string `json:"alert_id"`
	Timestamp int64  `json:"timestamp"`
	Resolved  bool   `json:"resolved,omitempty"`
}

type SystemMonitor struct {
	httpClient    *http.Client
	incidents     map[string][]Incident
	betterStackURL string
	hostname      string
	cpuLimit      float64
	memoryLimit   float64
	diskLimit     float64
	interval      int
}

func NewSystemMonitor(betterStackURL string, interval int, cpuLimit, memoryLimit, diskLimit float64) (*SystemMonitor, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %v", err)
	}

	return &SystemMonitor{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		incidents: map[string][]Incident{
			"cpu":    {},
			"memory": {},
			"disk":   {},
		},
		betterStackURL: betterStackURL,
		hostname:      hostname,
		cpuLimit:      cpuLimit,
		memoryLimit:   memoryLimit,
		diskLimit:     diskLimit,
		interval:      interval,
	}, nil
}

func (s *SystemMonitor) evaluateCPUIncident() (*Incident, error) {
	duration := float64(s.interval) / 10
	if duration < 5 {
		duration = 5
	}
	if duration > 60 {
		duration = 60
	}

	cpuPercent, err := cpu.Percent(time.Duration(duration)*time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %v", err)
	}

	if len(cpuPercent) == 0 {
		return nil, nil
	}

	log.Printf("CPU usage: %.2f%%\n", cpuPercent[0])
	if cpuPercent[0] > s.cpuLimit {
		return &Incident{
			Title:     fmt.Sprintf("CPU usage higher than %.0f%%! - %s", s.cpuLimit, s.hostname),
			Cause:     "High CPU usage",
			AlertID:   fmt.Sprintf("high-cpu-%s", s.hostname),
			Timestamp: time.Now().Unix(),
		}, nil
	}

	return nil, nil
}

func (s *SystemMonitor) evaluateMemoryIncident() (*Incident, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory stats: %v", err)
	}

	log.Printf("Memory usage: %.2f%% (Available: %d MB, Total: %d MB)\n",
		vmStat.UsedPercent,
		vmStat.Available/(1024*1024),
		vmStat.Total/(1024*1024))

	if vmStat.UsedPercent > s.memoryLimit {
		return &Incident{
			Title:     fmt.Sprintf("Memory usage higher than %.0f%%! - %s", s.memoryLimit, s.hostname),
			Cause:     "High memory usage",
			AlertID:   fmt.Sprintf("high-memory-%s", s.hostname),
			Timestamp: time.Now().Unix(),
		}, nil
	}

	return nil, nil
}

func (s *SystemMonitor) evaluateDiskIncident() ([]Incident, error) {
	var incidents []Incident

	// Check root partition
	usage, err := disk.Usage("/")
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage: %v", err)
	}

	log.Printf("Diskspace used /: %.2f%% (Free: %d MB, Total: %d MB)\n",
		usage.UsedPercent,
		usage.Free/(1024*1024),
		usage.Total/(1024*1024))

	if usage.UsedPercent > s.diskLimit {
		incidents = append(incidents, Incident{
			Title:     fmt.Sprintf("Root disk usage higher than %.0f%%! - %s", s.diskLimit, s.hostname),
			Cause:     "High disk usage",
			AlertID:   fmt.Sprintf("high-disk-%s", s.hostname),
			Timestamp: time.Now().Unix(),
		})
	}

	// Check mounted directories
	mounts, err := filepath.Glob("/mnt/*")
	if err != nil {
		return nil, fmt.Errorf("failed to list mounted directories: %v", err)
	}

	for _, mount := range mounts {
		usage, err := disk.Usage(mount)
		if err != nil {
			log.Printf("Failed to get disk usage for %s: %v\n", mount, err)
			continue
		}

		log.Printf("Diskspace used %s: %.2f%% (Free: %d MB, Total: %d MB)\n",
			mount,
			usage.UsedPercent,
			usage.Free/(1024*1024),
			usage.Total/(1024*1024))

		if usage.UsedPercent > s.diskLimit {
			incidents = append(incidents, Incident{
				Title:     fmt.Sprintf("%s disk usage higher than %.0f%%! - %s", mount, s.diskLimit, s.hostname),
				Cause:     "High disk usage",
				AlertID:   fmt.Sprintf("high-disk-%s", s.hostname),
				Timestamp: time.Now().Unix(),
			})
		}
	}

	return incidents, nil
}

func (s *SystemMonitor) createIncident(incident Incident) error {
	log.Printf("Triggering incident: %s\n", incident.Title)
	return s.sendIncident(incident)
}

func (s *SystemMonitor) resolveIncident(incident Incident) error {
	log.Printf("Resolving incident: %s\n", incident.Title)
	incident.Resolved = true
	return s.sendIncident(incident)
}

func (s *SystemMonitor) sendIncident(incident Incident) error {
	body, err := json.Marshal(incident)
	if err != nil {
		return fmt.Errorf("failed to marshal incident: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.betterStackURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Appwrite system-monitoring")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (s *SystemMonitor) processType(monitorType string, evaluate func() (interface{}, error)) error {
	incidents, err := evaluate()
	if err != nil {
		return fmt.Errorf("failed to evaluate %s: %v", monitorType, err)
	}

	if incidents == nil {
		if len(s.incidents[monitorType]) > 0 {
			log.Printf("Resolving active incident of type %s\n", monitorType)
			for _, incident := range s.incidents[monitorType] {
				if err := s.resolveIncident(incident); err != nil {
					log.Printf("Failed to resolve incident: %v\n", err)
				}
			}
			s.incidents[monitorType] = nil
		}
		return nil
	}

	if len(s.incidents[monitorType]) > 0 {
		log.Printf("Already have active incident of type '%s', skipping.\n", monitorType)
		return nil
	}

	switch i := incidents.(type) {
	case *Incident:
		if i != nil {
			if err := s.createIncident(*i); err != nil {
				return fmt.Errorf("failed to create incident: %v", err)
			}
			s.incidents[monitorType] = []Incident{*i}
		}
	case []Incident:
		for _, incident := range i {
			if err := s.createIncident(incident); err != nil {
				return fmt.Errorf("failed to create incident: %v", err)
			}
		}
		s.incidents[monitorType] = i
	}

	return nil
}

func (s *SystemMonitor) Start() {
	ticker := time.NewTicker(time.Duration(s.interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.processType("cpu", func() (interface{}, error) {
			return s.evaluateCPUIncident()
		}); err != nil {
			log.Printf("Error processing CPU metrics: %v\n", err)
		}

		if err := s.processType("memory", func() (interface{}, error) {
			return s.evaluateMemoryIncident()
		}); err != nil {
			log.Printf("Error processing memory metrics: %v\n", err)
		}

		if err := s.processType("disk", func() (interface{}, error) {
			return s.evaluateDiskIncident()
		}); err != nil {
			log.Printf("Error processing disk metrics: %v\n", err)
		}
	}
}

func main() {
	// Define command line flags
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
		log.Fatal("Error: BetterStack webhook URL is required")
	}

	// Validate ranges
	if *interval <= 0 {
		log.Fatal("Error: interval must be greater than 0")
	}
	if *cpuLimit < 0 || *cpuLimit > 100 {
		log.Fatal("Error: cpu-limit must be between 0 and 100")
	}
	if *memoryLimit < 0 || *memoryLimit > 100 {
		log.Fatal("Error: memory-limit must be between 0 and 100")
	}
	if *diskLimit < 0 || *diskLimit > 100 {
		log.Fatal("Error: disk-limit must be between 0 and 100")
	}

	monitor, err := NewSystemMonitor(*betterStackURL, *interval, *cpuLimit, *memoryLimit, *diskLimit)
	if err != nil {
		log.Fatalf("Failed to create system monitor: %v", err)
	}

	log.Printf("Starting monitoring with settings:")
	log.Printf("- Check interval: %d seconds", *interval)
	log.Printf("- CPU limit: %.1f%%", *cpuLimit)
	log.Printf("- Memory limit: %.1f%%", *memoryLimit)
	log.Printf("- Disk limit: %.1f%%", *diskLimit)

	monitor.Start()
} 