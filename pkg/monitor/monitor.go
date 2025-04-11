package monitor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/appwrite/monitoring/pkg"
)

// Metric represents a monitoring metric to be sent to BetterStack
type Metric struct {
	Title     string  `json:"title"`
	Cause     string  `json:"cause"`
	AlertID   string  `json:"alert_id"`
	Timestamp int64   `json:"timestamp"`
	Status    string  `json:"status"`
	Value     float64 `json:"value"`
	Limit     float64 `json:"limit"`
}

// SystemMonitor handles system resource monitoring and alerts
type SystemMonitor struct {
	httpClient     *http.Client
	betterStackURL string
	hostname       string
	cpuLimit       float64
	memoryLimit    float64
	diskLimit      float64
	interval       int
	log            *pkg.Logger
	
	// EMA tracking
	cpuEMA         float64
	memoryEMA      float64
	diskEMAs       map[string]float64 // Map to track EMAs for all disks (root and mounted)
	alpha          float64 // EMA smoothing factor
}

// NewSystemMonitor creates a new system monitor instance
func NewSystemMonitor(betterStackURL string, interval int, cpuLimit, memoryLimit, diskLimit float64) (*SystemMonitor, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %v", err)
	}

	// Calculate alpha based on interval to get roughly 5 minutes of smoothing
	// EMA formula: alpha = 2/(N+1) where N is the number of periods
	// For 5 minutes of smoothing with our interval: N = 300/interval
	N := float64(300) / float64(interval)
	alpha := 2.0 / (N + 1.0)

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
		log:            pkg.NewLogger(),
		diskEMAs:       make(map[string]float64), // Initialize the map for all disk EMAs
		alpha:          alpha,
	}, nil
}

// calculateEMA calculates the exponential moving average
func (s *SystemMonitor) calculateEMA(currentValue, previousEMA float64) float64 {
	return s.alpha*currentValue + (1-s.alpha)*previousEMA
}

// getStatus determines if a metric is passing or failing
func (s *SystemMonitor) getStatus(value, limit float64) string {
	if value > limit {
		return "fail"
	}
	return "pass"
}

// sendMetric sends a metric to the BetterStack monitoring endpoint
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

// Start begins the monitoring process
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

// runChecks executes all monitoring checks
func (s *SystemMonitor) runChecks() {
	if err := s.CheckCPU(); err != nil {
		s.log.Error("Error checking CPU: %v", err)
	}

	if err := s.CheckMemory(); err != nil {
		s.log.Error("Error checking memory: %v", err)
	}

	if err := s.CheckDisk(); err != nil {
		s.log.Error("Error checking disk: %v", err)
	}
} 