package monitor

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
)

// CheckMemory monitors memory usage and sends metrics
func (s *SystemMonitor) CheckMemory() error {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("failed to get memory stats: %v", err)
	}

	instantValue := vmStat.UsedPercent
	s.memoryEMA = s.calculateEMA(instantValue, s.memoryEMA)
	
	status := s.getStatus(s.memoryEMA, s.memoryLimit)
	if status == "fail" {
		s.log.Warn("Memory usage EMA %.2f%% exceeds limit of %.2f%% (instant: %.2f%%)", s.memoryEMA, s.memoryLimit, instantValue)
	} else {
		s.log.Log("Memory usage EMA: %.2f%% (limit: %.2f%%, instant: %.2f%%), Available: %d MB, Total: %d MB",
			s.memoryEMA,
			s.memoryLimit,
			instantValue,
			vmStat.Available/(1024*1024),
			vmStat.Total/(1024*1024))
	}

	metric := Metric{
		Title:     fmt.Sprintf("Memory Usage - %s", s.hostname),
		Cause:     "Memory monitoring check",
		AlertID:   fmt.Sprintf("memory-%s", s.hostname),
		Timestamp: time.Now().Unix(),
		Status:    status,
		Value:     s.memoryEMA,
		Limit:     s.memoryLimit,
	}

	return s.sendMetric(metric)
} 