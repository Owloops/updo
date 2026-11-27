package tui

import (
	"testing"
)

func TestDetailsManager_InitializeWidgets(t *testing.T) {
	dm := NewDetailsManager()
	dm.InitializeWidgets()

	if dm.TimingBreakdownWidget == nil {
		t.Error("TimingBreakdownWidget is nil")
	}
	if dm.LogsWidget == nil {
		t.Error("LogsWidget is nil")
	}
}
