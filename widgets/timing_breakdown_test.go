package widgets

import (
	"testing"
	"time"
)

func TestTimingBreakdown_calculateTotalDuration(t *testing.T) {
	tests := []struct {
		name      string
		timings   map[string]time.Duration
		wantTotal time.Duration
	}{
		{
			name:      "empty timings",
			timings:   map[string]time.Duration{},
			wantTotal: 0,
		},
		{
			name: "single timing",
			timings: map[string]time.Duration{
				"DNS": 100 * time.Millisecond,
			},
			wantTotal: 100 * time.Millisecond,
		},
		{
			name: "multiple timings",
			timings: map[string]time.Duration{
				"Wait":     50 * time.Millisecond,
				"DNS":      100 * time.Millisecond,
				"TCP":      150 * time.Millisecond,
				"TTFB":     200 * time.Millisecond,
				"Download": 500 * time.Millisecond,
			},
			wantTotal: 1000 * time.Millisecond,
		},
		{
			name: "zero duration mixed with non-zero",
			timings: map[string]time.Duration{
				"Wait":     0,
				"DNS":      100 * time.Millisecond,
				"TCP":      0,
				"TTFB":     200 * time.Millisecond,
				"Download": 0,
			},
			wantTotal: 300 * time.Millisecond,
		},
		{
			name: "very large durations",
			timings: map[string]time.Duration{
				"Wait":     time.Hour,
				"DNS":      2 * time.Hour,
				"Download": 3 * time.Hour,
			},
			wantTotal: 6 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := NewTimingBreakdown()
			tb.Timings = tt.timings

			got := tb.calculateTotalDuration()
			if got != tt.wantTotal {
				t.Errorf("calculateTotalDuration() = %v, want %v", got, tt.wantTotal)
			}
		})
	}
}

func TestTimingBreakdown_longestLabel(t *testing.T) {
	tb := NewTimingBreakdown()

	expected := 9
	got := tb.longestLabel()

	if got != expected {
		t.Errorf("longestLabel() = %d, want %d", got, expected)
	}
}

func TestTimingBreakdown_SetTimings(t *testing.T) {
	tests := []struct {
		name       string
		newTimings map[string]time.Duration
	}{
		{
			name:       "nil timings",
			newTimings: nil,
		},
		{
			name:       "empty timings",
			newTimings: map[string]time.Duration{},
		},
		{
			name: "normal timings",
			newTimings: map[string]time.Duration{
				"Wait": 100 * time.Millisecond,
				"DNS":  200 * time.Millisecond,
			},
		},
		{
			name: "unknown stages",
			newTimings: map[string]time.Duration{
				"Unknown1": 100 * time.Millisecond,
				"Unknown2": 200 * time.Millisecond,
			},
		},
		{
			name: "negative durations",
			newTimings: map[string]time.Duration{
				"Wait": -100 * time.Millisecond,
				"DNS":  -200 * time.Millisecond,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := NewTimingBreakdown()
			tb.SetTimings(tt.newTimings)

			if tb.Timings == nil && tt.newTimings != nil {
				t.Error("SetTimings() set Timings to nil when it shouldn't")
			}

			if tt.newTimings != nil {
				for k, v := range tt.newTimings {
					if tb.Timings[k] != v {
						t.Errorf("SetTimings() Timings[%s] = %v, want %v", k, tb.Timings[k], v)
					}
				}
			}
		})
	}
}

func TestTimingBreakdown_EdgeCases(t *testing.T) {
	t.Run("Draw with zero total duration", func(t *testing.T) {
		tb := NewTimingBreakdown()
		tb.Timings = map[string]time.Duration{
			"Wait": 0,
			"DNS":  0,
			"TCP":  0,
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Draw() panicked with zero durations: %v", r)
			}
		}()

		total := tb.calculateTotalDuration()
		if total != 0 {
			t.Errorf("Expected total duration to be 0, got %v", total)
		}
	})

	t.Run("Draw with single microsecond timing", func(t *testing.T) {
		tb := NewTimingBreakdown()
		tb.Timings = map[string]time.Duration{
			"DNS": time.Microsecond,
		}

		total := tb.calculateTotalDuration()
		if total != time.Microsecond {
			t.Errorf("Expected total duration to be 1µs, got %v", total)
		}
	})
}
