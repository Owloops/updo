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
			input:     "https://google.com",
			wantHTTPS: "https://google.com",
			wantHTTP:  "https://google.com",
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
		{
			name:            "redirect follow false accept false",
			statusCode:      301,
			responseBody:    "Moved Permanently",
			config:          NetworkConfig{FollowRedirects: false, AcceptRedirects: false, Timeout: 5 * time.Second},
			expectSuccess:   false,
			expectAssertion: true,
		},
		{
			name:            "redirect follow false accept true",
			statusCode:      301,
			responseBody:    "Moved Permanently",
			config:          NetworkConfig{FollowRedirects: false, AcceptRedirects: true, Timeout: 5 * time.Second},
			expectSuccess:   true,
			expectAssertion: true,
		},
		{
			name:            "redirect follow true accept false",
			statusCode:      301,
			responseBody:    "Moved Permanently",
			config:          NetworkConfig{FollowRedirects: true, AcceptRedirects: false, Timeout: 5 * time.Second},
			expectSuccess:   false,
			expectAssertion: true,
		},
		{
			name:            "redirect follow true accept true",
			statusCode:      301,
			responseBody:    "Moved Permanently",
			config:          NetworkConfig{FollowRedirects: true, AcceptRedirects: true, Timeout: 5 * time.Second},
			expectSuccess:   true,
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

func TestCheckWebsiteWithHostHeader(t *testing.T) {
	const wantHost = "virtual.example.com"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != wantHost {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(r.Host))
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := NetworkConfig{
		Timeout: 5 * time.Second,
		Headers: []string{"Host: " + wantHost},
	}

	result := CheckWebsite(server.URL, config)

	if !result.IsUp {
		t.Errorf("CheckWebsite() with Host header should succeed, got status %d body %q", result.StatusCode, result.ResponseBody)
	}
	if result.StatusCode != 200 {
		t.Errorf("CheckWebsite() StatusCode = %d, want 200", result.StatusCode)
	}
	if got := result.RequestHeaders.Get("Host"); got != wantHost {
		t.Errorf("RequestHeaders Host = %q, want %q", got, wantHost)
	}
}

func TestCheckWebsiteBodyLimit(t *testing.T) {
	tests := []struct {
		name           string
		bodySize       int
		bodySizeLimit  int64
		wantTruncated  bool
		wantBodyLength int
	}{
		{
			name:           "body under custom limit",
			bodySize:       500,
			bodySizeLimit:  1000,
			wantTruncated:  false,
			wantBodyLength: 500,
		},
		{
			name:           "body equals custom limit",
			bodySize:       1000,
			bodySizeLimit:  1000,
			wantTruncated:  false,
			wantBodyLength: 1000,
		},
		{
			name:           "body exceeds custom limit",
			bodySize:       2000,
			bodySizeLimit:  1000,
			wantTruncated:  true,
			wantBodyLength: 1000,
		},
		{
			name:           "zero BodySizeLimit means unlimited",
			bodySize:       2 * 1024 * 1024,
			bodySizeLimit:  0,
			wantTruncated:  false,
			wantBodyLength: 2 * 1024 * 1024,
		},
		{
			name:           "DefaultBodySizeLimit caps at 1 MiB",
			bodySize:       2 * 1024 * 1024,
			bodySizeLimit:  DefaultBodySizeLimit,
			wantTruncated:  true,
			wantBodyLength: 1 << 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := make([]byte, tt.bodySize)
			for i := range payload {
				payload[i] = 'x'
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				_, _ = w.Write(payload)
			}))
			defer server.Close()

			config := NetworkConfig{Timeout: 5 * time.Second, BodySizeLimit: tt.bodySizeLimit}
			result := CheckWebsite(server.URL, config)

			if result.ResponseTruncated != tt.wantTruncated {
				t.Errorf("ResponseTruncated = %v, want %v", result.ResponseTruncated, tt.wantTruncated)
			}
			if len(result.ResponseBody) != tt.wantBodyLength {
				t.Errorf("ResponseBody length = %d, want %d", len(result.ResponseBody), tt.wantBodyLength)
			}
		})
	}
}
