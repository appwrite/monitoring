package monitor

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
)

// CheckDisk monitors disk usage for root and mounted partitions
func (s *SystemMonitor) CheckDisk() error {
	// Check root partition
	rootPath := "/"
	usage, err := disk.Usage(rootPath)
	if err != nil {
		return fmt.Errorf("failed to get disk usage: %v", err)
	}

	instantValue := usage.UsedPercent
	
	// Calculate or update EMA for root disk
	if _, exists := s.diskEMAs[rootPath]; !exists {
		// Initialize EMA with current value if this is first check
		s.diskEMAs[rootPath] = instantValue
	}
	
	// Update EMA for root disk
	s.diskEMAs[rootPath] = s.calculateEMA(instantValue, s.diskEMAs[rootPath])
	
	rootEMA := s.diskEMAs[rootPath]
	status := s.getStatus(rootEMA, s.diskLimit)
	
	if status == "fail" {
		s.log.Warn("Root disk usage EMA %.2f%% exceeds limit of %.2f%% (instant: %.2f%%)", rootEMA, s.diskLimit, instantValue)
	} else {
		s.log.Log("Root disk usage EMA: %.2f%% (limit: %.2f%%, instant: %.2f%%), Free: %d MB, Total: %d MB",
			rootEMA,
			s.diskLimit,
			instantValue,
			usage.Free/(1024*1024),
			usage.Total/(1024*1024))
	}

	if err := s.sendMetric(Metric{
		Title:     fmt.Sprintf("Root Disk Usage - %s", s.hostname),
		Cause:     "Disk monitoring check",
		AlertID:   fmt.Sprintf("disk-root-%s", s.hostname),
		Timestamp: time.Now().Unix(),
		Status:    status,
		Value:     rootEMA,
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

		instantValue := usage.UsedPercent
		
		// Calculate or update EMA for this mount
		if _, exists := s.diskEMAs[mount]; !exists {
			// Initialize EMA with current value if this is first check
			s.diskEMAs[mount] = instantValue
		}
		
		// Update EMA for this mount
		s.diskEMAs[mount] = s.calculateEMA(instantValue, s.diskEMAs[mount])
		
		mountEMA := s.diskEMAs[mount]
		status := s.getStatus(mountEMA, s.diskLimit)
		
		if status == "fail" {
			s.log.Warn("Disk usage for %s EMA %.2f%% exceeds limit of %.2f%% (instant: %.2f%%)", mount, mountEMA, s.diskLimit, instantValue)
		} else {
			s.log.Log("Disk usage for %s EMA: %.2f%% (limit: %.2f%%, instant: %.2f%%), Free: %d MB, Total: %d MB",
				mount,
				mountEMA,
				s.diskLimit,
				instantValue,
				usage.Free/(1024*1024),
				usage.Total/(1024*1024))
		}

		if err := s.sendMetric(Metric{
			Title:     fmt.Sprintf("Disk Usage %s - %s", mount, s.hostname),
			Cause:     "Disk monitoring check",
			AlertID:   fmt.Sprintf("disk-%s-%s", filepath.Base(mount), s.hostname),
			Timestamp: time.Now().Unix(),
			Status:    status,
			Value:     mountEMA,
			Limit:     s.diskLimit,
		}); err != nil {
			return err
		}
	}

	return nil
} 