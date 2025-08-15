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

func TestSanitizeDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected time.Duration
	}{
		{"Valid positive duration", 100 * time.Millisecond, 100 * time.Millisecond},
		{"Valid zero duration", 0, 0},
		{"Negative duration", -100 * time.Millisecond, 0},
		{"Extreme negative duration", -2562047*time.Hour - 47*time.Minute, 0},
		{"Extreme positive duration", 2562047*time.Hour + 47*time.Minute, 0},
		{"Just under 24 hours", 23*time.Hour + 59*time.Minute + 59*time.Second, 23*time.Hour + 59*time.Minute + 59*time.Second},
		{"Exactly 24 hours", 24 * time.Hour, 0},
		{"Just over 24 hours", 24*time.Hour + 1*time.Nanosecond, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeDuration(tc.input)
			if result != tc.expected {
				t.Errorf("SanitizeDuration(%v) = %v, expected %v", tc.input, result, tc.expected)
			}
		})
	}
}
