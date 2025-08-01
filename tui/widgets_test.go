package tui

import (
	"testing"
	"time"

	"github.com/Owloops/updo/net"
	"github.com/gizak/termui/v3/widgets"
)

func TestDetailsManager_Creation(t *testing.T) {
	dm := NewDetailsManager()
	if dm == nil {
		t.Fatal("NewDetailsManager returned nil")
	}
}

func TestDetailsManager_UpdatePlotsData(t *testing.T) {
	dm := NewDetailsManager()
	dm.UptimePlot = &widgets.Plot{Data: [][]float64{{}}}
	dm.ResponseTimePlot = &widgets.Plot{Data: [][]float64{{}}}
	
	tests := []struct {
		name   string
		result net.WebsiteCheckResult
		width  int
	}{
		{
			name: "successful check",
			result: net.WebsiteCheckResult{
				IsUp:         true,
				ResponseTime: 250 * time.Millisecond,
			},
			width: 100,
		},
		{
			name: "failed check", 
			result: net.WebsiteCheckResult{
				IsUp:         false,
				ResponseTime: 0,
			},
			width: 100,
		},
		{
			name: "data truncation test",
			result: net.WebsiteCheckResult{
				IsUp:         true,
				ResponseTime: 100 * time.Millisecond,
			},
			width: 10,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeUptime := len(dm.UptimePlot.Data[0])
			beforeResponse := len(dm.ResponseTimePlot.Data[0])
			
			dm.updatePlotsData(tt.result, tt.width)
			
			if len(dm.UptimePlot.Data[0]) != beforeUptime+1 {
				t.Error("Uptime plot data not updated")
			}
			
			if len(dm.ResponseTimePlot.Data[0]) != beforeResponse+1 {
				t.Error("Response time plot data not updated")
			}
			
			maxLength := tt.width / 2
			if maxLength > 0 && len(dm.UptimePlot.Data[0]) > maxLength {
				t.Errorf("Data should be truncated to %d, got uptime: %d, response: %d", 
					maxLength, len(dm.UptimePlot.Data[0]), len(dm.ResponseTimePlot.Data[0]))
			}
		})
	}
}

func TestDetailsManager_DataTruncation(t *testing.T) {
	dm := NewDetailsManager()
	dm.UptimePlot = &widgets.Plot{Data: [][]float64{{}}}
	dm.ResponseTimePlot = &widgets.Plot{Data: [][]float64{{}}}
	
	width := 20
	maxLength := width / 2
	
	for i := 0; i < 25; i++ {
		dm.updatePlotsData(net.WebsiteCheckResult{
			IsUp:         i%2 == 0,
			ResponseTime: time.Duration(i*10) * time.Millisecond,
		}, width)
	}
	
	if len(dm.UptimePlot.Data[0]) > maxLength {
		t.Errorf("Uptime data not truncated: got %d, want <= %d", 
			len(dm.UptimePlot.Data[0]), maxLength)
	}
	
	if len(dm.ResponseTimePlot.Data[0]) > maxLength {
		t.Errorf("Response data not truncated: got %d, want <= %d", 
			len(dm.ResponseTimePlot.Data[0]), maxLength)
	}
}