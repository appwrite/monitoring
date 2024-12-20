package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type Metric struct {
	Title     string  `json:"title"`
	Cause     string  `json:"cause"`
	AlertID   string  `json:"alert_id"`
	Timestamp int64   `json:"timestamp"`
	Status    string  `json:"status"`
	Value     float64 `json:"value"`
	Limit     float64 `json:"limit"`
}

type SystemMonitor struct {
	httpClient     *http.Client
	betterStackURL string
	hostname       string
	cpuLimit       float64
	memoryLimit    float64
	diskLimit      float64
	interval       int
	log            *Logger
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
		betterStackURL: betterStackURL,
		hostname:       hostname,
		cpuLimit:       cpuLimit,
		memoryLimit:    memoryLimit,
		diskLimit:      diskLimit,
		interval:       interval,
		log:            New(),
	}, nil
}

func (s *SystemMonitor) checkCPU() error {
	duration := float64(s.interval) / 10
	if duration < 5 {
		duration = 5
	}
	if duration > 60 {
		duration = 60
	}

	cpuPercent, err := cpu.Percent(time.Duration(duration)*time.Second, false)
	if err != nil {
		return fmt.Errorf("failed to get CPU usage: %v", err)
	}

	if len(cpuPercent) == 0 {
		return nil
	}

	value := cpuPercent[0]
	status := s.getStatus(value, s.cpuLimit)
	if status == "fail" {
		s.log.Warn("CPU usage %.2f%% exceeds limit of %.2f%%", value, s.cpuLimit)
	} else {
		s.log.Log("CPU usage: %.2f%% (limit: %.2f%%)", value, s.cpuLimit)
	}
	
	metric := Metric{
		Title:     fmt.Sprintf("CPU Usage - %s", s.hostname),
		Cause:     "CPU monitoring check",
		AlertID:   fmt.Sprintf("cpu-%s", s.hostname),
		Timestamp: time.Now().Unix(),
		Status:    status,
		Value:     value,
		Limit:     s.cpuLimit,
	}

	return s.sendMetric(metric)
}

func (s *SystemMonitor) checkMemory() error {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("failed to get memory stats: %v", err)
	}

	value := vmStat.UsedPercent
	status := s.getStatus(value, s.memoryLimit)
	if status == "fail" {
		s.log.Warn("Memory usage %.2f%% exceeds limit of %.2f%%", value, s.memoryLimit)
	} else {
		s.log.Log("Memory usage: %.2f%% (limit: %.2f%%), Available: %d MB, Total: %d MB",
			value,
			s.memoryLimit,
			vmStat.Available/(1024*1024),
			vmStat.Total/(1024*1024))
	}

	metric := Metric{
		Title:     fmt.Sprintf("Memory Usage - %s", s.hostname),
		Cause:     "Memory monitoring check",
		AlertID:   fmt.Sprintf("memory-%s", s.hostname),
		Timestamp: time.Now().Unix(),
		Status:    status,
		Value:     value,
		Limit:     s.memoryLimit,
	}

	return s.sendMetric(metric)
}

func (s *SystemMonitor) checkDisk() error {
	// Check root partition
	usage, err := disk.Usage("/")
	if err != nil {
		return fmt.Errorf("failed to get disk usage: %v", err)
	}

	value := usage.UsedPercent
	status := s.getStatus(value, s.diskLimit)
	if status == "fail" {
		s.log.Warn("Root disk usage %.2f%% exceeds limit of %.2f%%", value, s.diskLimit)
	} else {
		s.log.Log("Root disk usage: %.2f%% (limit: %.2f%%), Free: %d MB, Total: %d MB",
			value,
			s.diskLimit,
			usage.Free/(1024*1024),
			usage.Total/(1024*1024))
	}

	if err := s.sendMetric(Metric{
		Title:     fmt.Sprintf("Root Disk Usage - %s", s.hostname),
		Cause:     "Disk monitoring check",
		AlertID:   fmt.Sprintf("disk-root-%s", s.hostname),
		Timestamp: time.Now().Unix(),
		Status:    status,
		Value:     value,
		Limit:     s.diskLimit,
	}); err != nil {
		return err
	}

	// Check mounted directories
	mounts, err := filepath.Glob("/mnt/*")
	if err != nil {
		return fmt.Errorf("failed to list mounted directories: %v", err)
	}

	for _, mount := range mounts {
		usage, err := disk.Usage(mount)
		if err != nil {
			s.log.Error("Failed to get disk usage for %s: %v", mount, err)
			continue
		}

		value := usage.UsedPercent
		status := s.getStatus(value, s.diskLimit)
		if status == "fail" {
			s.log.Warn("Disk usage for %s %.2f%% exceeds limit of %.2f%%", mount, value, s.diskLimit)
		} else {
			s.log.Log("Disk usage for %s: %.2f%% (limit: %.2f%%), Free: %d MB, Total: %d MB",
				mount,
				value,
				s.diskLimit,
				usage.Free/(1024*1024),
				usage.Total/(1024*1024))
		}

		if err := s.sendMetric(Metric{
			Title:     fmt.Sprintf("Disk Usage %s - %s", mount, s.hostname),
			Cause:     "Disk monitoring check",
			AlertID:   fmt.Sprintf("disk-%s-%s", filepath.Base(mount), s.hostname),
			Timestamp: time.Now().Unix(),
			Status:    status,
			Value:     value,
			Limit:     s.diskLimit,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *SystemMonitor) getStatus(value, limit float64) string {
	if value > limit {
		return "fail"
	}
	return "pass"
}

func (s *SystemMonitor) sendMetric(metric Metric) error {
	body, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.betterStackURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Appwrite Resource Monitoring")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	s.log.Log("Response Status: %s", resp.Status)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (s *SystemMonitor) Start() {
	ticker := time.NewTicker(time.Duration(s.interval) * time.Second)
	defer ticker.Stop()

	// Initial check
	s.runChecks()

	// Periodic checks
	for range ticker.C {
		s.runChecks()
	}
}

func (s *SystemMonitor) runChecks() {
	if err := s.checkCPU(); err != nil {
		s.log.Error("Error checking CPU: %v", err)
	}

	if err := s.checkMemory(); err != nil {
		s.log.Error("Error checking memory: %v", err)
	}

	if err := s.checkDisk(); err != nil {
		s.log.Error("Error checking disk: %v", err)
	}
}

func main() {
	log := New()

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

	monitor, err := NewSystemMonitor(*betterStackURL, *interval, *cpuLimit, *memoryLimit, *diskLimit)
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