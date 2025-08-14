package net

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsUrl(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid domain",
			input: "https://example.com",
			want:  true,
		},
		{
			name:  "domain with port",
			input: "https://example.com:8080",
			want:  true,
		},
		{
			name:  "IPv4 address",
			input: "https://192.168.1.1",
			want:  true,
		},
		{
			name:  "IPv4 with port",
			input: "https://192.168.1.1:8080",
			want:  true,
		},
		{
			name:  "IPv6 address",
			input: "https://[::1]",
			want:  true,
		},
		{
			name:  "localhost hostname",
			input: "https://localhost",
			want:  true,
		},
		{
			name:  "localhost with port",
			input: "https://localhost:3000",
			want:  true,
		},
		{
			name:  "single-label hostname",
			input: "https://myserver",
			want:  true,
		},
		{
			name:  "missing protocol",
			input: "example.com",
			want:  false,
		},
		{
			name:  "invalid URL format",
			input: "not a url",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUrl(tt.input)
			if got != tt.want {
				t.Errorf("isUrl(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsIPAddress(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "IPv4 address",
			input: "https://192.168.1.1",
			want:  true,
		},
		{
			name:  "IPv6 address",
			input: "https://[::1]",
			want:  true,
		},
		{
			name:  "domain name",
			input: "https://example.com",
			want:  false,
		},
		{
			name:  "localhost",
			input: "https://localhost",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIPAddress(tt.input)
			if got != tt.want {
				t.Errorf("isIPAddress(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestAutoDetectProtocol(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantHTTPS string
		wantHTTP  string
	}{
		{
			name:      "add protocol to domain",
			input:     "example.com",
			wantHTTPS: "https://example.com",
			wantHTTP:  "http://example.com",
		},
		{
			name:      "preserve existing protocol",
			input:     "https://example.com",
			wantHTTPS: "https://example.com",
			wantHTTP:  "https://example.com",
		},
		{
			name:      "add protocol to IP with port",
			input:     "192.168.1.1:8080",
			wantHTTPS: "https://192.168.1.1:8080",
			wantHTTP:  "http://192.168.1.1:8080",
		},
		{
			name:      "add protocol to localhost",
			input:     "localhost:3000",
			wantHTTPS: "https://localhost:3000",
			wantHTTP:  "http://localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AutoDetectProtocol(tt.input)
			if got != tt.wantHTTPS && got != tt.wantHTTP {
				t.Errorf("AutoDetectProtocol(%q) = %v, want %v or %v", tt.input, got, tt.wantHTTPS, tt.wantHTTP)
			}
		})
	}
}

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name     string
		headers  []string
		expected map[string]string
	}{
		{
			name:     "valid headers",
			headers:  []string{"Content-Type: application/json", "Authorization: Bearer token"},
			expected: map[string]string{"Content-Type": "application/json", "Authorization": "Bearer token"},
		},
		{
			name:     "headers with spaces",
			headers:  []string{" Content-Type : application/json "},
			expected: map[string]string{"Content-Type": "application/json"},
		},
		{
			name:     "malformed header skipped",
			headers:  []string{"valid: header", "invalid-header", "another: valid"},
			expected: map[string]string{"valid": "header", "another": "valid"},
		},
		{
			name:     "empty headers",
			headers:  []string{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseHeaders(tt.headers)
			if len(got) != len(tt.expected) {
				t.Errorf("parseHeaders() returned %d headers, want %d", len(got), len(tt.expected))
			}
			for key, expectedValue := range tt.expected {
				if gotValue, exists := got[key]; !exists {
					t.Errorf("parseHeaders() missing key %q", key)
				} else if gotValue != expectedValue {
					t.Errorf("parseHeaders() key %q = %q, want %q", key, gotValue, expectedValue)
				}
			}
		})
	}
}

func TestFormatURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "add https to domain",
			input: "example.com",
			want:  "https://example.com",
		},
		{
			name:  "preserve existing protocol",
			input: "http://example.com",
			want:  "http://example.com",
		},
		{
			name:  "trim whitespace",
			input: "  example.com  ",
			want:  "https://example.com",
		},
		{
			name:  "localhost with port",
			input: "localhost:3000",
			want:  "https://localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatURL(tt.input)
			if err != nil {
				t.Errorf("formatURL(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("formatURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCheckWebsite(t *testing.T) {
	tests := []struct {
		name            string
		statusCode      int
		responseBody    string
		config          NetworkConfig
		expectSuccess   bool
		expectAssertion bool
	}{
		{
			name:            "successful request",
			statusCode:      200,
			responseBody:    "OK",
			config:          NetworkConfig{Timeout: 5 * time.Second},
			expectSuccess:   true,
			expectAssertion: true,
		},
		{
			name:            "server error",
			statusCode:      500,
			responseBody:    "Internal Server Error",
			config:          NetworkConfig{Timeout: 5 * time.Second},
			expectSuccess:   false,
			expectAssertion: true,
		},
		{
			name:            "assertion success",
			statusCode:      200,
			responseBody:    "Hello World",
			config:          NetworkConfig{AssertText: "Hello", Timeout: 5 * time.Second},
			expectSuccess:   true,
			expectAssertion: true,
		},
		{
			name:            "assertion failure",
			statusCode:      200,
			responseBody:    "Hello World",
			config:          NetworkConfig{AssertText: "Goodbye", Timeout: 5 * time.Second},
			expectSuccess:   false,
			expectAssertion: false,
		},
		{
			name:            "should fail inverted",
			statusCode:      500,
			responseBody:    "Error",
			config:          NetworkConfig{ShouldFail: true, Timeout: 5 * time.Second},
			expectSuccess:   true,
			expectAssertion: true,
		},
		{
			name:            "redirect without accept redirects",
			statusCode:      301,
			responseBody:    "Moved Permanently",
			config:          NetworkConfig{AcceptRedirects: false, Timeout: 5 * time.Second},
			expectSuccess:   false,
			expectAssertion: true,
		},
		{
			name:            "redirect with accept redirects",
			statusCode:      301,
			responseBody:    "Moved Permanently",
			config:          NetworkConfig{AcceptRedirects: true, Timeout: 5 * time.Second},
			expectSuccess:   true,
			expectAssertion: true,
		},
		{
			name:            "302 redirect with accept redirects",
			statusCode:      302,
			responseBody:    "Found",
			config:          NetworkConfig{AcceptRedirects: true, Timeout: 5 * time.Second},
			expectSuccess:   true,
			expectAssertion: true,
		},
		{
			name:            "304 not modified with accept redirects",
			statusCode:      304,
			responseBody:    "Not Modified",
			config:          NetworkConfig{AcceptRedirects: true, Timeout: 5 * time.Second},
			expectSuccess:   true,
			expectAssertion: true,
		},
		{
			name:            "redirect with accept redirects and assertion failure",
			statusCode:      301,
			responseBody:    "Moved Permanently",
			config:          NetworkConfig{AcceptRedirects: true, AssertText: "Not Found", Timeout: 5 * time.Second},
			expectSuccess:   false,
			expectAssertion: false,
		},
		{
			name:            "redirect with accept redirects and should fail",
			statusCode:      301,
			responseBody:    "Moved Permanently",
			config:          NetworkConfig{AcceptRedirects: true, ShouldFail: true, Timeout: 5 * time.Second},
			expectSuccess:   false,
			expectAssertion: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			result := CheckWebsite(server.URL, tt.config)

			if result.IsUp != tt.expectSuccess {
				t.Errorf("CheckWebsite() IsUp = %v, want %v", result.IsUp, tt.expectSuccess)
			}
			if result.AssertionPassed != tt.expectAssertion {
				t.Errorf("CheckWebsite() AssertionPassed = %v, want %v", result.AssertionPassed, tt.expectAssertion)
			}
			if result.StatusCode != tt.statusCode {
				t.Errorf("CheckWebsite() StatusCode = %d, want %d", result.StatusCode, tt.statusCode)
			}
			if result.ResponseTime <= 0 {
				t.Error("CheckWebsite() ResponseTime should be positive")
			}
		})
	}
}

func TestCheckWebsiteWithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(401)
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte("Authorized"))
	}))
	defer server.Close()

	config := NetworkConfig{
		Timeout: 5 * time.Second,
		Headers: []string{"Authorization: Bearer test-token", "Content-Type: application/json"},
	}

	result := CheckWebsite(server.URL, config)

	if !result.IsUp {
		t.Error("CheckWebsite() with headers should succeed")
	}
	if result.StatusCode != 200 {
		t.Errorf("CheckWebsite() StatusCode = %d, want 200", result.StatusCode)
	}
}
