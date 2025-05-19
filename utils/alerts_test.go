package utils

import (
	"testing"
)

func TestHandleAlertsLogic(t *testing.T) {
	tests := []struct {
		name         string
		isUp         bool
		initialSent  bool
		expectedSent bool
	}{
		{
			name:         "Site goes down for the first time",
			isUp:         false,
			initialSent:  false,
			expectedSent: true,
		},
		{
			name:         "Site is still down",
			isUp:         false,
			initialSent:  true,
			expectedSent: true,
		},
		{
			name:         "Site comes back up",
			isUp:         true,
			initialSent:  true,
			expectedSent: false,
		},
		{
			name:         "Site is still up",
			isUp:         true,
			initialSent:  false,
			expectedSent: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			alertSent := tc.initialSent

			HandleAlerts(tc.isUp, &alertSent)

			if alertSent != tc.expectedSent {
				t.Errorf("Expected alertSent to be: %v, got: %v", tc.expectedSent, alertSent)
			}
		})
	}
}
