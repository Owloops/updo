package httputil

import "testing"

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name     string
		headers  []string
		expected map[string]string
	}{
		{
			name:     "valid headers",
			headers:  []string{"Content-Type: application/json", "Authorization: Bearer token"},
			expected: map[string]string{"Content-Type": "application/json", "Authorization": "Bearer token"},
		},
		{
			name:     "headers with spaces",
			headers:  []string{" Content-Type : application/json "},
			expected: map[string]string{"Content-Type": "application/json"},
		},
		{
			name:     "malformed header skipped",
			headers:  []string{"valid: header", "invalid-header", "another: valid"},
			expected: map[string]string{"valid": "header", "another": "valid"},
		},
		{
			name:     "empty headers",
			headers:  []string{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseHeaders(tt.headers)
			if len(got) != len(tt.expected) {
				t.Errorf("ParseHeaders() returned %d headers, want %d", len(got), len(tt.expected))
			}
			for key, expectedValue := range tt.expected {
				if gotValue, exists := got[key]; !exists {
					t.Errorf("ParseHeaders() missing key %q", key)
				} else if gotValue != expectedValue {
					t.Errorf("ParseHeaders() key %q = %q, want %q", key, gotValue, expectedValue)
				}
			}
		})
	}
}
