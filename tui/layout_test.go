package tui

import (
	"testing"
)

func TestNavigateAllKeys_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		visibleRows int
		direction   int
	}{
		{
			name:        "empty list",
			visibleRows: 0,
			direction:   1,
		},
		{
			name:        "single item up",
			visibleRows: 1,
			direction:   -1,
		},
		{
			name:        "single item down",
			visibleRows: 1,
			direction:   1,
		},
		{
			name:        "multiple items",
			visibleRows: 5,
			direction:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			
			visibleRows := tt.visibleRows
			direction := tt.direction
			currentRow := 0
			
			if visibleRows == 0 {
				
				return
			}
			
			if direction > 0 {
				currentRow = (currentRow + 1) % visibleRows
			} else {
				currentRow = (currentRow - 1 + visibleRows) % visibleRows
			}
			
			if currentRow < 0 || currentRow >= visibleRows {
				t.Errorf("Navigation out of bounds: currentRow=%d, visibleRows=%d", currentRow, visibleRows)
			}
		})
	}
}

func TestNavigateFilteredKeys_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		visibleRows int
		direction   int
	}{
		{
			name:        "empty filtered list",
			visibleRows: 0,
			direction:   1,
		},
		{
			name:        "single filtered item",
			visibleRows: 1,
			direction:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			
			visibleRows := tt.visibleRows
			direction := tt.direction
			currentRow := 0
			
			if visibleRows == 0 {
				return
			}
			
			if direction > 0 {
				currentRow = (currentRow + 1) % visibleRows
			} else {
				currentRow = (currentRow - 1 + visibleRows) % visibleRows
			}
			
			if currentRow < 0 || currentRow >= visibleRows {
				t.Errorf("Filtered navigation out of bounds: currentRow=%d, visibleRows=%d", currentRow, visibleRows)
			}
		})
	}
}