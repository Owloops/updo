package net

import (
	"testing"
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
		name  string
		input string
		want  string
	}{
		{
			name:  "add protocol to domain",
			input: "example.com",
			want:  "https://example.com",
		},
		{
			name:  "preserve existing protocol",
			input: "https://example.com",
			want:  "https://example.com",
		},
		{
			name:  "add protocol to IP with port",
			input: "192.168.1.1:8080",
			want:  "https://192.168.1.1:8080",
		},
		{
			name:  "add protocol to localhost",
			input: "localhost:3000",
			want:  "https://localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AutoDetectProtocol(tt.input)
			if got != tt.want {
				t.Errorf("AutoDetectProtocol(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
