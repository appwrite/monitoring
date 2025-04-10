package main

import (
	"fmt"
	"math"
	"testing"
)

func TestEMACalculation(t *testing.T) {
	// Create a monitor with a 60-second interval (alpha will be 2/(5+1) = 0.333)
	monitor, err := NewSystemMonitor("http://test.com", 60, 90.0, 90.0, 90.0)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	// Test case 1: Steady high usage
	t.Run("SteadyHighUsage", func(t *testing.T) {
		// Simulate 10 measurements of 95% CPU usage
		for i := 0; i < 10; i++ {
			monitor.cpuEMA = monitor.calculateEMA(95.0, monitor.cpuEMA)
		}
		
		// EMA should be close to 95% after 10 measurements
		if monitor.cpuEMA < 93.0 || monitor.cpuEMA > 96.0 {
			t.Errorf("Expected EMA to be around 95%% after steady high usage, got %.2f%%", monitor.cpuEMA)
		}
	})

	// Test case 2: Single spike
	t.Run("SingleSpike", func(t *testing.T) {
		// Reset EMA
		monitor.cpuEMA = 50.0
		
		// Single spike to 100%
		monitor.cpuEMA = monitor.calculateEMA(100.0, monitor.cpuEMA)
		
		// EMA should be much lower than the spike
		if monitor.cpuEMA > 70.0 {
			t.Errorf("Expected EMA to be significantly lower than spike (100%%), got %.2f%%", monitor.cpuEMA)
		}
	})

	// Test case 3: Gradual increase
	t.Run("GradualIncrease", func(t *testing.T) {
		// Reset EMA
		monitor.cpuEMA = 50.0
		
		// Simulate gradual increase from 50% to 90%
		values := []float64{60.0, 70.0, 80.0, 90.0}
		for _, v := range values {
			monitor.cpuEMA = monitor.calculateEMA(v, monitor.cpuEMA)
		}
		
		// EMA should be between 70% and 90%
		if monitor.cpuEMA < 70.0 || monitor.cpuEMA > 90.0 {
			t.Errorf("Expected EMA to be between 70%% and 90%% after gradual increase, got %.2f%%", monitor.cpuEMA)
		}
	})
}

func TestEMASmoothing(t *testing.T) {
	// Create a monitor with a 60-second interval
	monitor, err := NewSystemMonitor("http://test.com", 60, 90.0, 90.0, 90.0)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	// Test case: Alternating high and low values
	t.Run("AlternatingValues", func(t *testing.T) {
		// Reset EMA
		monitor.cpuEMA = 50.0
		
		// Simulate alternating between 20% and 100%
		for i := 0; i < 20; i++ {
			var value float64
			if i%2 == 0 {
				value = 20.0
			} else {
				value = 100.0
			}
			monitor.cpuEMA = monitor.calculateEMA(value, monitor.cpuEMA)
		}
		
		// EMA should be around 60% (smoothing out the extremes)
		if monitor.cpuEMA < 50.0 || monitor.cpuEMA > 70.0 {
			t.Errorf("Expected EMA to be around 60%% after alternating values, got %.2f%%", monitor.cpuEMA)
		}
	})
}

func TestEMAResponseTime(t *testing.T) {
	// Create a monitor with a 60-second interval
	monitor, err := NewSystemMonitor("http://test.com", 60, 90.0, 90.0, 90.0)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	// Test case: Measure how quickly EMA responds to sustained change
	t.Run("ResponseTime", func(t *testing.T) {
		// Reset EMA
		monitor.cpuEMA = 50.0
		
		// Simulate sustained high usage
		measurements := 0
		for monitor.cpuEMA < 80.0 {
			monitor.cpuEMA = monitor.calculateEMA(100.0, monitor.cpuEMA)
			measurements++
			
			// Prevent infinite loop
			if measurements > 20 {
				t.Fatal("EMA took too long to respond to sustained high usage")
			}
		}
		
		t.Logf("EMA reached 80%% after %d measurements", measurements)
	})
}

func TestEMADifferentIntervals(t *testing.T) {
	// Test different intervals to verify alpha calculation
	testCases := []struct {
		interval       int
		expectedAlpha  float64
	}{
		{60, 0.333},  // 5 minutes = 5 periods
		{30, 0.182},  // 5 minutes = 10 periods
		{15, 0.095},  // 5 minutes = 20 periods
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Interval%d", tc.interval), func(t *testing.T) {
			monitor, err := NewSystemMonitor("http://test.com", tc.interval, 90.0, 90.0, 90.0)
			if err != nil {
				t.Fatalf("Failed to create monitor: %v", err)
			}
			
			// Allow for small floating point differences
			if math.Abs(monitor.alpha - tc.expectedAlpha) > 0.001 {
				t.Errorf("Expected alpha %.3f for interval %d, got %.3f", 
					tc.expectedAlpha, tc.interval, monitor.alpha)
			}
		})
	}
} 