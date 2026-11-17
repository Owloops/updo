package monitor

import (
	"testing"

	"github.com/Owloops/updo/stats"
)

func TestExitCode(t *testing.T) {
	tests := []struct {
		name    string
		summary stats.Summary
		want    int
	}{
		{name: "all up", summary: stats.Summary{AllUp: true}, want: 0},
		{name: "target down", summary: stats.Summary{AllUp: false}, want: _exitFailure},
		{name: "interrupted with target down", summary: stats.Summary{AllUp: false, Interrupted: true}, want: 0},
		{name: "interrupted with all up", summary: stats.Summary{AllUp: true, Interrupted: true}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := exitCode(tt.summary); got != tt.want {
				t.Errorf("exitCode(%+v) = %d, want %d", tt.summary, got, tt.want)
			}
		})
	}
}
