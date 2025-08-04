package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	prompb "buf.build/gen/go/prometheus/prometheus/protocolbuffers/go"
	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
)

func TestWriteClient(t *testing.T) {
	cfg := NewConfig()
	client := NewWriteClient(cfg)

	if client == nil || client.config.ServerURL != cfg.ServerURL ||
		client.samples == nil || client.httpClient == nil {
		t.Fatal("WriteClient not properly initialized")
	}

	target := config.Target{Name: "test", URL: "https://example.com"}
	result := net.WebsiteCheckResult{URL: target.URL, IsUp: true, StatusCode: 200}

	initialCount := len(client.samples)
	client.AddCheck(target, result, "us-east-1")

	if len(client.samples) <= initialCount {
		t.Error("AddCheck failed to add samples")
	}

	zeroTimestamp := time.Time{}.UnixMilli()
	for _, series := range client.samples {
		for _, sample := range series.Samples {
			if sample.Timestamp != zeroTimestamp {
				t.Errorf("Expected placeholder timestamp %d, got %d", zeroTimestamp, sample.Timestamp)
			}
		}
	}
}

func TestSSLExpiry(t *testing.T) {
	client := NewWriteClient(NewConfig())
	target := config.Target{Name: "ssl-test", URL: "https://example.com"}

	tests := []struct {
		days int
		add  bool
	}{
		{30, true}, {0, true}, {-1, false},
	}

	for _, tt := range tests {
		initial := len(client.samples)
		client.AddSSLExpiry(target, tt.days)
		added := len(client.samples) > initial
		if added != tt.add {
			t.Errorf("SSL expiry days=%d: expected add=%v, got=%v", tt.days, tt.add, added)
		}
	}
}

func TestNormalizeTimestamps(t *testing.T) {
	client := NewWriteClient(NewConfig())
	samples := []*prompb.TimeSeries{
		{
			Labels:  []*prompb.Label{{Name: "__name__", Value: "test1"}},
			Samples: []*prompb.Sample{{Timestamp: 0, Value: 1.0}, {Timestamp: 0, Value: 2.0}},
		},
		{
			Labels:  []*prompb.Label{{Name: "__name__", Value: "test2"}},
			Samples: []*prompb.Sample{{Timestamp: 0, Value: 3.0}},
		},
		nil,
		{Labels: []*prompb.Label{{Name: "__name__", Value: "empty"}}, Samples: []*prompb.Sample{}},
		{
			Labels:  []*prompb.Label{{Name: "__name__", Value: "mixed"}},
			Samples: []*prompb.Sample{nil, {Timestamp: 0, Value: 4.0}},
		},
	}

	before := time.Now().UnixMilli()
	normalized := client.normalizeTimestamps(samples)
	after := time.Now().UnixMilli()

	if len(normalized) != 3 {
		t.Errorf("Expected 3 valid series, got %d", len(normalized))
	}

	var batchTime int64
	for i, series := range normalized {
		for j, sample := range series.Samples {
			if sample.Timestamp < before || sample.Timestamp > after {
				t.Errorf("Series %d sample %d: timestamp out of range", i, j)
			}
			if batchTime == 0 {
				batchTime = sample.Timestamp
			} else if sample.Timestamp != batchTime {
				t.Errorf("Inconsistent timestamps: expected %d, got %d", batchTime, sample.Timestamp)
			}
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	client := NewWriteClient(NewConfig())
	target := config.Target{Name: "concurrent", URL: "https://example.com"}
	result := net.WebsiteCheckResult{URL: target.URL, IsUp: true}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			client.AddCheck(target, result, "region")
		}()
		go func() {
			defer wg.Done()
			client.AddSSLExpiry(target, 30)
		}()
	}
	wg.Wait()

	if len(client.samples) == 0 {
		t.Error("No samples added from concurrent operations")
	}
}

func TestHTTPRequests(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		auth   bool
		header bool
		expect bool
	}{
		{"success", 200, "", false, false, true},
		{"error", 400, "out of order", false, false, false},
		{"auth", 200, "", true, false, true},
		{"headers", 200, "", false, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.auth {
					if user, pass, ok := r.BasicAuth(); !ok || user != "user" || pass != "pass" {
						t.Error("Auth failed")
					}
				}
				if tt.header && r.Header.Get("X-Test") != "value" {
					t.Error("Header missing")
				}
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			cfg := NewConfig()
			cfg.ServerURL = server.URL
			if tt.auth {
				cfg.Username, cfg.Password = "user", "pass"
			}
			if tt.header {
				cfg.Headers = map[string]string{"X-Test": "value"}
			}

			client := NewWriteClient(cfg)
			err := client.doRequest([]byte("data"))
			success := err == nil

			if success != tt.expect {
				t.Errorf("Expected success=%v, got err=%v", tt.expect, err)
			}
			if !tt.expect && err != nil && !strings.Contains(err.Error(), tt.body) {
				t.Errorf("Error should contain response body")
			}
		})
	}
}
