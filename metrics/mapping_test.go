package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
)

func TestMapTargetLabels(t *testing.T) {
	target := config.Target{Name: "service", URL: "https://example.com"}
	result := net.WebsiteCheckResult{URL: target.URL, IsUp: true, StatusCode: 200}

	labels := MapTargetLabels(target, result, "us-east-1")
	expected := map[string]string{"name": "service", "url": "https://example.com", "region": "us-east-1"}

	for key, want := range expected {
		if got := labels[key]; got != want {
			t.Errorf("Label %s: expected %s, got %s", key, want, got)
		}
	}
}

func TestMapSeries(t *testing.T) {
	labels := map[string]string{"name": "test", "url": "https://example.com", "": "empty", "empty": ""}
	pbLabels := MapSeries("target_up", labels)

	if len(pbLabels) != 3 {
		t.Errorf("Expected 3 labels after filtering, got %d", len(pbLabels))
	}

	labelMap := make(map[string]string)
	for _, label := range pbLabels {
		labelMap[label.Name] = label.Value
		if label.Name == "" || label.Value == "" {
			t.Error("Found empty label after filtering")
		}
	}

	if labelMap["__name__"] != "updo_target_up" {
		t.Errorf("Expected __name__=updo_target_up, got %s", labelMap["__name__"])
	}

	for i := 1; i < len(pbLabels); i++ {
		if pbLabels[i-1].Name >= pbLabels[i].Name {
			t.Error("Labels not sorted properly")
		}
	}
}

func TestConvertCheckToTimeSeries(t *testing.T) {
	tests := []struct {
		name     string
		target   config.Target
		result   net.WebsiteCheckResult
		expected map[string]float64
	}{
		{
			"up_target",
			config.Target{Name: "test", URL: "https://example.com"},
			net.WebsiteCheckResult{URL: "https://example.com", IsUp: true, StatusCode: 200, ResponseTime: 100 * time.Millisecond},
			map[string]float64{"target_up": 1.0, "response_time_seconds": 0.1, "http_status_code_total": 1.0},
		},
		{
			"down_target",
			config.Target{Name: "failing", URL: "https://broken.com"},
			net.WebsiteCheckResult{URL: "https://broken.com", IsUp: false, StatusCode: 500},
			map[string]float64{"target_up": 0.0, "http_status_code_total": 1.0},
		},
		{
			"with_assertion",
			config.Target{Name: "assert", URL: "https://api.com", AssertText: "success"},
			net.WebsiteCheckResult{URL: "https://api.com", IsUp: true, StatusCode: 200, AssertText: "success", AssertionPassed: true},
			map[string]float64{"target_up": 1.0, "assertion_passed": 1.0, "http_status_code_total": 1.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeSeries := ConvertCheckToTimeSeries(tt.target, tt.result, "", time.Now())

			metrics := make(map[string]float64)
			for _, series := range timeSeries {
				for _, label := range series.Labels {
					if label.Name == _nameLbl {
						metricName := strings.TrimPrefix(label.Value, "updo_")
						if len(series.Samples) > 0 {
							metrics[metricName] = series.Samples[0].Value
						}
					}
				}
			}

			for metric, expectedValue := range tt.expected {
				if value, exists := metrics[metric]; !exists {
					t.Errorf("Expected metric %s not found", metric)
				} else if value != expectedValue {
					t.Errorf("Metric %s: expected %f, got %f", metric, expectedValue, value)
				}
			}
		})
	}
}

func TestConvertWithTraceInfo(t *testing.T) {
	target := config.Target{Name: "traced", URL: "https://example.com"}
	result := net.WebsiteCheckResult{
		URL: target.URL, IsUp: true, StatusCode: 200,
		TraceInfo: &net.HttpTraceInfo{
			DNSLookup: 10 * time.Millisecond, TCPConnection: 20 * time.Millisecond,
			TimeToFirstByte: 50 * time.Millisecond, DownloadDuration: 20 * time.Millisecond,
		},
	}

	timeSeries := ConvertCheckToTimeSeries(target, result, "", time.Now())
	expectedMetrics := []string{"target_up", "http_status_code_total", "dns_lookup_seconds", "tcp_connection_seconds", "time_to_first_byte_seconds", "download_duration_seconds"}

	found := make(map[string]bool)
	for _, series := range timeSeries {
		for _, label := range series.Labels {
			if label.Name == "__name__" {
				metricName := strings.TrimPrefix(label.Value, "updo_")
				found[metricName] = true
			}
		}
	}

	for _, metric := range expectedMetrics {
		if !found[metric] {
			t.Errorf("Expected timing metric %s not found", metric)
		}
	}
}

func TestConvertSSLExpiryToTimeSeries(t *testing.T) {
	target := config.Target{Name: "ssl-test", URL: "https://secure.com"}
	tests := []struct {
		days   int
		expect bool
		value  float64
	}{
		{30, true, 30.0}, {0, true, 0.0}, {-1, false, 0},
	}

	for _, tt := range tests {
		result := ConvertSSLExpiryToTimeSeries(target, tt.days, time.Now())

		if tt.expect {
			if result == nil {
				t.Errorf("Expected SSL series for %d days", tt.days)
				continue
			}
			if len(result.Samples) != 1 || result.Samples[0].Value != tt.value {
				t.Errorf("SSL days=%d: expected value=%f, got=%f", tt.days, tt.value, result.Samples[0].Value)
			}

			var foundSSLMetric bool
			for _, label := range result.Labels {
				if label.Name == "__name__" && strings.Contains(label.Value, "ssl_cert_expiry_days") {
					foundSSLMetric = true
				}
			}
			if !foundSSLMetric {
				t.Error("SSL metric name not found")
			}
		} else if result != nil {
			t.Errorf("Expected no SSL series for %d days", tt.days)
		}
	}
}
