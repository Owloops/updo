package stats

import (
	"math"
	"testing"
	"time"

	"github.com/Owloops/updo/net"
)

func TestNewMonitor(t *testing.T) {
	monitor, err := NewMonitor()
	if err != nil {
		t.Fatalf("NewMonitor failed: %v", err)
	}

	if monitor.ChecksCount != 0 {
		t.Errorf("Expected ChecksCount=0, got %d", monitor.ChecksCount)
	}
	if monitor.MinResponseTime != time.Duration(math.MaxInt64) {
		t.Errorf("Expected MinResponseTime=MaxInt64, got %v", monitor.MinResponseTime)
	}
	if monitor.TDigest == nil {
		t.Error("Expected TDigest to be initialized")
	}
}

func TestMonitor_AddResult(t *testing.T) {
	tests := []struct {
		name            string
		results         []net.WebsiteCheckResult
		expectedChecks  int
		expectedSuccess int
		expectedIsUp    bool
		expectedMinRT   time.Duration
		expectedMaxRT   time.Duration
	}{
		{
			name: "single successful result",
			results: []net.WebsiteCheckResult{
				{IsUp: true, ResponseTime: 100 * time.Millisecond, StatusCode: 200},
			},
			expectedChecks:  1,
			expectedSuccess: 1,
			expectedIsUp:    true,
			expectedMinRT:   100 * time.Millisecond,
			expectedMaxRT:   100 * time.Millisecond,
		},
		{
			name: "single failed result",
			results: []net.WebsiteCheckResult{
				{IsUp: false, ResponseTime: 5000 * time.Millisecond, StatusCode: 500},
			},
			expectedChecks:  1,
			expectedSuccess: 0,
			expectedIsUp:    false,
			expectedMinRT:   5000 * time.Millisecond,
			expectedMaxRT:   5000 * time.Millisecond,
		},
		{
			name: "mixed results",
			results: []net.WebsiteCheckResult{
				{IsUp: true, ResponseTime: 100 * time.Millisecond, StatusCode: 200},
				{IsUp: false, ResponseTime: 200 * time.Millisecond, StatusCode: 500},
				{IsUp: true, ResponseTime: 50 * time.Millisecond, StatusCode: 200},
			},
			expectedChecks:  3,
			expectedSuccess: 2,
			expectedIsUp:    true,
			expectedMinRT:   50 * time.Millisecond,
			expectedMaxRT:   200 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor, _ := NewMonitor()

			for _, result := range tt.results {
				monitor.AddResult(result)
			}

			if monitor.ChecksCount != tt.expectedChecks {
				t.Errorf("Expected ChecksCount=%d, got %d", tt.expectedChecks, monitor.ChecksCount)
			}
			if monitor.SuccessCount != tt.expectedSuccess {
				t.Errorf("Expected SuccessCount=%d, got %d", tt.expectedSuccess, monitor.SuccessCount)
			}
			if monitor.IsUp != tt.expectedIsUp {
				t.Errorf("Expected IsUp=%v, got %v", tt.expectedIsUp, monitor.IsUp)
			}
			if monitor.MinResponseTime != tt.expectedMinRT {
				t.Errorf("Expected MinResponseTime=%v, got %v", tt.expectedMinRT, monitor.MinResponseTime)
			}
			if monitor.MaxResponseTime != tt.expectedMaxRT {
				t.Errorf("Expected MaxResponseTime=%v, got %v", tt.expectedMaxRT, monitor.MaxResponseTime)
			}
		})
	}
}

func TestMonitor_GetStats(t *testing.T) {
	tests := []struct {
		name           string
		results        []net.WebsiteCheckResult
		expectedUptime float64
		expectedAvgRT  time.Duration
		expectP95      bool
		expectStdDev   bool
	}{
		{
			name:           "no results",
			results:        []net.WebsiteCheckResult{},
			expectedUptime: 0,
			expectedAvgRT:  0,
			expectP95:      false,
			expectStdDev:   false,
		},
		{
			name: "single result",
			results: []net.WebsiteCheckResult{
				{IsUp: true, ResponseTime: 100 * time.Millisecond},
			},
			expectedAvgRT: 100 * time.Millisecond,
			expectP95:     false,
			expectStdDev:  false,
		},
		{
			name: "multiple results for P95",
			results: []net.WebsiteCheckResult{
				{IsUp: true, ResponseTime: 100 * time.Millisecond},
				{IsUp: true, ResponseTime: 200 * time.Millisecond},
				{IsUp: true, ResponseTime: 150 * time.Millisecond},
			},
			expectedAvgRT: 150 * time.Millisecond,
			expectP95:     true,
			expectStdDev:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor, _ := NewMonitor()

			for _, result := range tt.results {
				monitor.AddResult(result)
			}

			stats := monitor.GetStats()

			if stats.ChecksCount != len(tt.results) {
				t.Errorf("Expected ChecksCount=%d, got %d", len(tt.results), stats.ChecksCount)
			}

			if len(tt.results) > 0 && stats.AvgResponseTime != tt.expectedAvgRT {
				t.Errorf("Expected AvgResponseTime=%v, got %v", tt.expectedAvgRT, stats.AvgResponseTime)
			}

			if tt.expectP95 && stats.P95 == 0 {
				t.Error("Expected P95 to be calculated")
			}
			if !tt.expectP95 && stats.P95 != 0 {
				t.Error("Expected P95 to be zero")
			}

			if tt.expectStdDev && stats.StdDev == 0 {
				t.Error("Expected StdDev to be calculated")
			}
			if !tt.expectStdDev && stats.StdDev != 0 {
				t.Error("Expected StdDev to be zero")
			}
		})
	}
}

func TestMonitor_UptimeCalculation(t *testing.T) {
	monitor, _ := NewMonitor()

	monitor.AddResult(net.WebsiteCheckResult{IsUp: true, ResponseTime: 100 * time.Millisecond})

	time.Sleep(1 * time.Millisecond)

	stats := monitor.GetStats()

	if stats.UptimePercent <= 0 {
		t.Errorf("Expected positive uptime, got %f", stats.UptimePercent)
	}
	if stats.UptimePercent > 100 {
		t.Errorf("Expected uptime <= 100%%, got %f", stats.UptimePercent)
	}
}

func TestMonitor_ResponseTimeStats(t *testing.T) {
	monitor, _ := NewMonitor()

	monitor.AddResult(net.WebsiteCheckResult{IsUp: true, ResponseTime: 100 * time.Millisecond})
	monitor.AddResult(net.WebsiteCheckResult{IsUp: true, ResponseTime: 200 * time.Millisecond})
	monitor.AddResult(net.WebsiteCheckResult{IsUp: true, ResponseTime: 300 * time.Millisecond})

	stats := monitor.GetStats()

	expectedAvg := 200 * time.Millisecond
	if stats.AvgResponseTime != expectedAvg {
		t.Errorf("Expected avg=%v, got %v", expectedAvg, stats.AvgResponseTime)
	}

	if stats.MinResponseTime != 100*time.Millisecond {
		t.Errorf("Expected min=100ms, got %v", stats.MinResponseTime)
	}

	if stats.MaxResponseTime != 300*time.Millisecond {
		t.Errorf("Expected max=300ms, got %v", stats.MaxResponseTime)
	}

	if stats.StdDev == 0 {
		t.Error("Expected non-zero standard deviation")
	}
}

func TestMonitor_SuccessRate(t *testing.T) {
	tests := []struct {
		name           string
		successResults int
		failResults    int
		expectedRate   int
	}{
		{"all success", 5, 0, 5},
		{"all fail", 0, 5, 0},
		{"mixed", 3, 2, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor, _ := NewMonitor()

			for range tt.successResults {
				monitor.AddResult(net.WebsiteCheckResult{IsUp: true, ResponseTime: 100 * time.Millisecond})
			}

			for range tt.failResults {
				monitor.AddResult(net.WebsiteCheckResult{IsUp: false, ResponseTime: 100 * time.Millisecond})
			}

			stats := monitor.GetStats()

			if stats.SuccessCount != tt.expectedRate {
				t.Errorf("Expected SuccessCount=%d, got %d", tt.expectedRate, stats.SuccessCount)
			}

			totalChecks := tt.successResults + tt.failResults
			if stats.ChecksCount != totalChecks {
				t.Errorf("Expected ChecksCount=%d, got %d", totalChecks, stats.ChecksCount)
			}
		})
	}
}

func TestMonitor_EdgeCases(t *testing.T) {
	t.Run("zero response time", func(t *testing.T) {
		monitor, _ := NewMonitor()
		monitor.AddResult(net.WebsiteCheckResult{IsUp: true, ResponseTime: 0})

		stats := monitor.GetStats()
		if stats.MinResponseTime != 0 {
			t.Errorf("Expected MinResponseTime=0, got %v", stats.MinResponseTime)
		}
		if stats.AvgResponseTime != 0 {
			t.Errorf("Expected AvgResponseTime=0, got %v", stats.AvgResponseTime)
		}
	})

	t.Run("very large response time", func(t *testing.T) {
		monitor, _ := NewMonitor()
		largeTime := 30 * time.Second
		monitor.AddResult(net.WebsiteCheckResult{IsUp: true, ResponseTime: largeTime})

		stats := monitor.GetStats()
		if stats.MaxResponseTime != largeTime {
			t.Errorf("Expected MaxResponseTime=%v, got %v", largeTime, stats.MaxResponseTime)
		}
	})
}
