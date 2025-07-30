package tui

import (
	"testing"
	"time"

	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
)

func TestDataStore_TargetData(t *testing.T) {
	tests := []struct {
		name   string
		target config.Target
		region string
		data   TargetData
	}{
		{
			name:   "local_target",
			target: config.Target{Name: "web-server", URL: "https://example.com"},
			region: "",
			data: TargetData{
				Target: config.Target{Name: "web-server", URL: "https://example.com"},
				Region: "",
			},
		},
		{
			name:   "regional_target",
			target: config.Target{Name: "web-server", URL: "https://example.com"},
			region: "us-east-1",
			data: TargetData{
				Target: config.Target{Name: "web-server", URL: "https://example.com"},
				Region: "us-east-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataStore := NewDataStore()
			var key TargetKey
			if tt.region != "" {
				key = NewRegionalTargetKey(tt.target, tt.region)
			} else {
				key = NewLocalTargetKey(tt.target)
			}

			dataStore.UpdateTargetData(key, tt.data)
			retrieved, exists := dataStore.GetTargetData(key)

			if !exists {
				t.Error("Target data should exist after update")
			}
			if retrieved.Target.Name != tt.target.Name {
				t.Errorf("Retrieved target name = %s, want %s", retrieved.Target.Name, tt.target.Name)
			}
			if retrieved.Region != tt.region {
				t.Errorf("Retrieved region = %s, want %s", retrieved.Region, tt.region)
			}
		})
	}
}

func TestDataStore_PlotData(t *testing.T) {
	tests := []struct {
		name       string
		result     net.WebsiteCheckResult
		termWidth  int
		expectUp   float64
		expectResp bool
	}{
		{
			name: "up_result",
			result: net.WebsiteCheckResult{
				URL:          "https://example.com",
				IsUp:         true,
				ResponseTime: 150 * time.Millisecond,
			},
			termWidth:  100,
			expectUp:   1.0,
			expectResp: true,
		},
		{
			name: "down_result",
			result: net.WebsiteCheckResult{
				URL:          "https://example.com",
				IsUp:         false,
				ResponseTime: 0,
			},
			termWidth:  100,
			expectUp:   0.0,
			expectResp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataStore := NewDataStore()
			target := config.Target{Name: "web-server", URL: "https://example.com"}
			key := NewLocalTargetKey(target)

			dataStore.UpdatePlotData(key, tt.result, tt.termWidth)
			history, exists := dataStore.GetPlotData(key)

			if !exists {
				t.Error("Plot data should exist after update")
			}
			if len(history.UptimeData) != 1 || history.UptimeData[0] != tt.expectUp {
				t.Errorf("Uptime data = %v, want [%f]", history.UptimeData, tt.expectUp)
			}
			if tt.expectResp && len(history.ResponseTimeData) < 2 {
				t.Error("Response time data should have at least 2 entries")
			}
		})
	}
}

func TestDataStore_SSLData(t *testing.T) {
	tests := []struct {
		name string
		url  string
		days int
	}{
		{"valid_ssl", "https://example.com", 30},
		{"expiring_ssl", "https://expiring.com", 7},
		{"expired_ssl", "https://expired.com", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataStore := NewDataStore()

			dataStore.UpdateSSLData(tt.url, tt.days)
			retrieved, exists := dataStore.GetSSLData(tt.url)

			if !exists {
				t.Error("SSL data should exist after update")
			}
			if retrieved != tt.days {
				t.Errorf("SSL days = %d, want %d", retrieved, tt.days)
			}
		})
	}
}

func TestDataStore_Validation(t *testing.T) {
	tests := []struct {
		name     string
		key      TargetKey
		data     TargetData
		expected bool
	}{
		{
			name:     "valid_local",
			key:      NewLocalTargetKey(config.Target{Name: "test", URL: "https://test.com"}),
			data:     TargetData{Target: config.Target{Name: "test", URL: "https://test.com"}, Region: ""},
			expected: true,
		},
		{
			name:     "valid_regional",
			key:      NewRegionalTargetKey(config.Target{Name: "test", URL: "https://test.com"}, "us-east-1"),
			data:     TargetData{Target: config.Target{Name: "test", URL: "https://test.com"}, Region: "us-east-1"},
			expected: true,
		},
		{
			name:     "invalid_region_mismatch",
			key:      NewRegionalTargetKey(config.Target{Name: "test", URL: "https://test.com"}, "us-east-1"),
			data:     TargetData{Target: config.Target{Name: "test", URL: "https://test.com"}, Region: "us-west-2"},
			expected: false,
		},
		{
			name:     "invalid_name_mismatch",
			key:      NewLocalTargetKey(config.Target{Name: "test1", URL: "https://test.com"}),
			data:     TargetData{Target: config.Target{Name: "test2", URL: "https://test.com"}, Region: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataStore := NewDataStore()
			result := dataStore.ValidateDataConsistency(tt.key, tt.data)
			if result != tt.expected {
				t.Errorf("Validation result = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDataStore_DataIsolation(t *testing.T) {
	dataStore := NewDataStore()

	target1 := config.Target{Name: "target1", URL: "https://example1.com"}
	target2 := config.Target{Name: "target2", URL: "https://example2.com"}

	key1 := NewLocalTargetKey(target1)
	key2 := NewLocalTargetKey(target2)

	data1 := TargetData{Target: target1, Region: ""}
	data2 := TargetData{Target: target2, Region: ""}

	dataStore.UpdateTargetData(key1, data1)
	dataStore.UpdateTargetData(key2, data2)

	retrieved1, _ := dataStore.GetTargetData(key1)
	retrieved2, _ := dataStore.GetTargetData(key2)

	if retrieved1.Target.Name == retrieved2.Target.Name {
		t.Error("Data should be isolated between different targets")
	}
}

func TestDataStore_GetAllKeys(t *testing.T) {
	dataStore := NewDataStore()
	target1 := config.Target{Name: "target1", URL: "https://example1.com"}
	target2 := config.Target{Name: "target2", URL: "https://example2.com"}

	key1 := NewLocalTargetKey(target1)
	key2 := NewRegionalTargetKey(target2, "us-east-1")

	data1 := TargetData{Target: target1, Region: ""}
	data2 := TargetData{Target: target2, Region: "us-east-1"}

	dataStore.UpdateTargetData(key1, data1)
	dataStore.UpdateTargetData(key2, data2)

	keys := dataStore.GetAllTargetKeys()

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}
}
