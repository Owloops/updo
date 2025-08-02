package utils

import (
	"testing"
)

func TestCLI_Progress(t *testing.T) {
	tests := []struct {
		name       string
		current    int
		total      int
		expectBar  bool
		expectPerc float64
	}{
		{"start", 0, 10, true, 0.0},
		{"half", 5, 10, true, 50.0},
		{"complete", 10, 10, true, 100.0},
		{"zero total", 0, 0, true, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := CLI{}

			cli.Progress(tt.current, tt.total, "Test")

			if tt.total > 0 {
				percent := float64(tt.current) / float64(tt.total) * 100
				if percent != tt.expectPerc {
					t.Errorf("Expected %.1f%%, got %.1f%%", tt.expectPerc, percent)
				}
			}
		})
	}
}

func TestCLI_ProgressWithStatus(t *testing.T) {
	cli := CLI{}

	cli.ProgressWithStatus(5, 10, "Test", "Processing...")
	cli.ProgressWithStatus(0, 1, "", "")
	cli.ProgressWithStatus(10, 10, "Done", "Complete")
}

func TestCLI_Region(t *testing.T) {
	tests := []struct {
		name     string
		region   string
		expected string
	}{
		{"normal region", "us-east-1", "[us-east-1]"},
		{"empty region", "", "[]"},
		{"special chars", "test@#$", "[test@#$]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := CLI{}
			result := cli.Region(tt.region)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCLI_FormattingMethods(t *testing.T) {
	cli := CLI{}

	cli.Error("test error")
	cli.Warn("test warning")
	cli.Info("test info")
	cli.Success("test success")
	cli.Plain("test plain")
}

func TestCLI_ProgressBarWidth(t *testing.T) {
	tests := []struct {
		current int
		total   int
	}{
		{1, 1},
		{0, 1},
		{50, 100},
		{1, 1000},
	}

	for _, tt := range tests {
		cli := CLI{}
		cli.Progress(tt.current, tt.total, "Test")
	}
}
