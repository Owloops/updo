package main

import (
	"context"
	"testing"
	"time"
)

func TestHandleRequest(t *testing.T) {
	tests := []struct {
		name           string
		request        CheckRequest
		expectError    bool
		expectSuccess  bool
		expectSSLCheck bool
	}{
		{
			name: "valid HTTP request",
			request: CheckRequest{
				URL:     "http://httpbin.org/status/200",
				Timeout: 10,
			},
			expectError:    false,
			expectSuccess:  true,
			expectSSLCheck: false,
		},
		{
			name: "valid HTTPS request",
			request: CheckRequest{
				URL:     "https://httpbin.org/status/200",
				Timeout: 10,
			},
			expectError:    false,
			expectSuccess:  true,
			expectSSLCheck: true,
		},
		{
			name: "missing URL",
			request: CheckRequest{
				Timeout: 10,
			},
			expectError:   true,
			expectSuccess: false,
		},
		{
			name: "default timeout applied",
			request: CheckRequest{
				URL:     "http://httpbin.org/status/200",
				Timeout: 0,
			},
			expectError:   false,
			expectSuccess: true,
		},
		{
			name: "timeout capping",
			request: CheckRequest{
				URL:     "http://httpbin.org/status/200",
				Timeout: 30,
			},
			expectError:   false,
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			resp, err := handleRequest(ctx, tt.request)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				if resp.Success != tt.expectSuccess {
					t.Errorf("Expected Success=%v, got %v", tt.expectSuccess, resp.Success)
				}
				if resp.Region == "" {
					t.Error("Region should not be empty")
				}
				if tt.expectSSLCheck && resp.SSLExpiry == nil {
					t.Error("Expected SSL expiry check for HTTPS URL")
				}
				if !tt.expectSSLCheck && resp.SSLExpiry != nil {
					t.Error("Did not expect SSL expiry check for HTTP URL")
				}
				if resp.ResponseTimeMs <= 0 && tt.expectSuccess {
					t.Error("Response time should be positive for successful requests")
				}
			}
		})
	}
}

func TestIsHTTPS(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://example.com", true},
		{"HTTPS://EXAMPLE.COM", true},
		{"http://example.com", false},
		{"ftp://example.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := isHTTPS(tt.url)
			if got != tt.want {
				t.Errorf("isHTTPS(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestCheckSSLExpiry(t *testing.T) {
	resp := CheckResponse{}

	checkSSLExpiry("https://google.com", &resp)

	if resp.SSLExpiry == nil {
		t.Error("Expected SSL expiry to be set for valid HTTPS site")
	} else if *resp.SSLExpiry < 0 {
		t.Errorf("Expected positive SSL expiry days, got %d", *resp.SSLExpiry)
	}
}

func TestTimeoutHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := CheckRequest{
		URL:     "http://httpbin.org/delay/1",
		Timeout: 10,
	}

	start := time.Now()
	resp, err := handleRequest(ctx, req)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if duration > 3*time.Second {
		t.Errorf("Request took too long: %v", duration)
	}

	if resp.Region == "" {
		t.Error("Region should not be empty")
	}
}
