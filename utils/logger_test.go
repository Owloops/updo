package utils

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
)

func TestLogMetrics(t *testing.T) {
	tests := []struct {
		name          string
		stats         *stats.Stats
		url           string
		region        []string
		expectP95     bool
		expectSuccess float64
	}{
		{"nil stats", nil, "https://example.com", nil, false, 0},
		{"basic stats", &stats.Stats{ChecksCount: 10, SuccessCount: 8}, "https://example.com", nil, false, 80.0},
		{"with region", &stats.Stats{ChecksCount: 5, SuccessCount: 5}, "https://example.com", []string{"us-east-1"}, false, 100.0},
		{"with P95", &stats.Stats{ChecksCount: 5, SuccessCount: 5, P95: 90 * time.Millisecond}, "https://example.com", nil, true, 100.0},
		{"zero checks", &stats.Stats{ChecksCount: 0, SuccessCount: 0}, "https://example.com", nil, false, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.stats == nil {
				LogMetrics(tt.stats, tt.url, tt.region...)
				return
			}

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			LogMetrics(tt.stats, tt.url, tt.region...)

			_ = w.Close()
			os.Stdout = oldStdout

			buf := make([]byte, 1024)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			var result map[string]interface{}
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Fatalf("Invalid JSON: %v", err)
			}

			if result["type"] != "metrics" {
				t.Errorf("Expected type=metrics, got %v", result["type"])
			}

			if result["success_percent"] != tt.expectSuccess {
				t.Errorf("Expected success_percent=%v, got %v", tt.expectSuccess, result["success_percent"])
			}

			_, hasP95 := result["p95_response_time_ms"]
			if tt.expectP95 != hasP95 {
				t.Errorf("Expected P95 present=%v, got %v", tt.expectP95, hasP95)
			}

			if len(tt.region) > 0 && tt.region[0] != "" {
				if result["region"] != tt.region[0] {
					t.Errorf("Expected region=%s, got %v", tt.region[0], result["region"])
				}
			}
		})
	}
}

func TestLogCheck(t *testing.T) {
	result := net.WebsiteCheckResult{
		URL:        "https://example.com",
		StatusCode: 200,
		IsUp:       true,
		Method:     "GET",
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	LogCheck(result, 1, "json", "us-east-1")

	_ = w.Close()
	os.Stdout = oldStdout

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if parsed["type"] != "check" {
		t.Errorf("Expected type=check, got %v", parsed["type"])
	}
	if parsed["success"] != true {
		t.Errorf("Expected success=true, got %v", parsed["success"])
	}
	if parsed["region"] != "us-east-1" {
		t.Errorf("Expected region=us-east-1, got %v", parsed["region"])
	}
}

func TestLogError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		expectErr bool
	}{
		{"with error", errors.New("timeout"), true},
		{"without error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			LogError("https://example.com", "test message", tt.err)

			_ = w.Close()
			os.Stderr = oldStderr

			buf := make([]byte, 1024)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			var result map[string]interface{}
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Fatalf("Invalid JSON: %v", err)
			}

			if result["type"] != "error" {
				t.Errorf("Expected type=error, got %v", result["type"])
			}

			_, hasError := result["error"]
			if tt.expectErr != hasError {
				t.Errorf("Expected error field present=%v, got %v", tt.expectErr, hasError)
			}
		})
	}
}

func TestLogWarning(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	LogWarning("https://example.com", "test warning", "us-east-1")

	_ = w.Close()
	os.Stderr = oldStderr

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if result["type"] != "warning" {
		t.Errorf("Expected type=warning, got %v", result["type"])
	}
	if result["level"] != "warning" {
		t.Errorf("Expected level=warning, got %v", result["level"])
	}
}
