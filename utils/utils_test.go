package utils

import (
	"testing"
	"time"
)

func TestBoolToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected float64
	}{
		{"True value", true, 1.0},
		{"False value", false, 0.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := BoolToFloat64(tc.input)
			if result != tc.expected {
				t.Errorf("BoolToFloat64(%v) = %v, expected %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestFormatDurationMillisecond(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{"Zero duration", 0 * time.Millisecond, "0 ms"},
		{"100 milliseconds", 100 * time.Millisecond, "100 ms"},
		{"1 second", 1 * time.Second, "1000 ms"},
		{"1.5 seconds", 1*time.Second + 500*time.Millisecond, "1500 ms"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatDurationMillisecond(tc.input)
			if result != tc.expected {
				t.Errorf("FormatDurationMillisecond(%v) = %v, expected %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestFormatDurationMinute(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{"Zero duration", 0 * time.Second, "0s"},
		{"10 seconds", 10 * time.Second, "10s"},
		{"1 minute", 1 * time.Minute, "1m0s"},
		{"1 hour 30 minutes", 1*time.Hour + 30*time.Minute, "1h30m0s"},
		{"Truncate to nearest second", 1*time.Minute + 499*time.Millisecond, "1m0s"},
		{"Truncate to nearest second (down)", 1*time.Minute + 500*time.Millisecond, "1m0s"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatDurationMinute(tc.input)
			if result != tc.expected {
				t.Errorf("FormatDurationMinute(%v) = %v, expected %v", tc.input, result, tc.expected)
			}
		})
	}
}
