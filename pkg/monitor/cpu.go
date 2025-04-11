package monitor

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

// CheckCPU monitors CPU usage and sends metrics
func (s *SystemMonitor) CheckCPU() error {
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

	// Calculate EMA for CPU usage
	instantValue := cpuPercent[0]
	s.cpuEMA = s.calculateEMA(instantValue, s.cpuEMA)
	
	status := s.getStatus(s.cpuEMA, s.cpuLimit)
	if status == "fail" {
		s.log.Warn("CPU usage EMA %.2f%% exceeds limit of %.2f%% (instant: %.2f%%)", s.cpuEMA, s.cpuLimit, instantValue)
	} else {
		s.log.Log("CPU usage EMA: %.2f%% (limit: %.2f%%, instant: %.2f%%)", s.cpuEMA, s.cpuLimit, instantValue)
	}
	
	metric := Metric{
		Title:     fmt.Sprintf("CPU Usage - %s", s.hostname),
		Cause:     "CPU monitoring check",
		AlertID:   fmt.Sprintf("cpu-%s", s.hostname),
		Timestamp: time.Now().Unix(),
		Status:    status,
		Value:     s.cpuEMA,
		Limit:     s.cpuLimit,
	}

	return s.sendMetric(metric)
} 