package tui

import (
	"testing"

	"github.com/Owloops/updo/config"
)

func TestTargetKey_Creation(t *testing.T) {
	tests := []struct {
		name           string
		target         config.Target
		region         string
		expectedString string
		expectedLocal  bool
		expectedRegion bool
	}{
		{
			name:           "local_key",
			target:         config.Target{Name: "web-server", URL: "https://example.com"},
			region:         "",
			expectedString: "web-server",
			expectedLocal:  true,
			expectedRegion: false,
		},
		{
			name:           "regional_key",
			target:         config.Target{Name: "web-server", URL: "https://example.com"},
			region:         "us-east-1",
			expectedString: "web-server@us-east-1",
			expectedLocal:  false,
			expectedRegion: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var key TargetKey
			if tt.region != "" {
				key = NewRegionalTargetKey(tt.target, tt.region)
			} else {
				key = NewLocalTargetKey(tt.target)
			}

			if key.String() != tt.expectedString {
				t.Errorf("Key string = %q, want %q", key.String(), tt.expectedString)
			}
			if key.IsLocal() != tt.expectedLocal {
				t.Errorf("IsLocal() = %v, want %v", key.IsLocal(), tt.expectedLocal)
			}
			if key.IsRegional() != tt.expectedRegion {
				t.Errorf("IsRegional() = %v, want %v", key.IsRegional(), tt.expectedRegion)
			}
		})
	}
}

func TestTargetKey_Validation(t *testing.T) {
	tests := []struct {
		name      string
		key       TargetKey
		wantError bool
	}{
		{
			name:      "valid_local",
			key:       TargetKey{TargetName: "test", Region: ""},
			wantError: false,
		},
		{
			name:      "valid_regional",
			key:       TargetKey{TargetName: "test", Region: "us-east-1"},
			wantError: false,
		},
		{
			name:      "invalid_empty_name",
			key:       TargetKey{TargetName: "", Region: "us-east-1"},
			wantError: true,
		},
		{
			name:      "invalid_empty_name_local",
			key:       TargetKey{TargetName: "", Region: ""},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.key.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestParseTargetKey(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedKey   TargetKey
		expectedError bool
	}{
		{
			name:          "local_key",
			input:         "web-server",
			expectedKey:   TargetKey{TargetName: "web-server", Region: ""},
			expectedError: false,
		},
		{
			name:          "regional_key",
			input:         "web-server@us-east-1",
			expectedKey:   TargetKey{TargetName: "web-server", Region: "us-east-1"},
			expectedError: false,
		},
		{
			name:          "empty_input",
			input:         "",
			expectedKey:   TargetKey{},
			expectedError: true,
		},
		{
			name:          "multiple_at_symbols",
			input:         "web@server@us-east-1",
			expectedKey:   TargetKey{TargetName: "web@server", Region: "us-east-1"},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := ParseTargetKey(tt.input)

			if (err != nil) != tt.expectedError {
				t.Errorf("ParseTargetKey() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			if !tt.expectedError {
				if key.TargetName != tt.expectedKey.TargetName {
					t.Errorf("TargetName = %s, want %s", key.TargetName, tt.expectedKey.TargetName)
				}
				if key.Region != tt.expectedKey.Region {
					t.Errorf("Region = %s, want %s", key.Region, tt.expectedKey.Region)
				}
			}
		})
	}
}

func TestTargetKey_Equality(t *testing.T) {
	target := config.Target{Name: "web-server", URL: "https://example.com"}

	localKey1 := NewLocalTargetKey(target)
	localKey2 := NewLocalTargetKey(target)
	regionalKey := NewRegionalTargetKey(target, "us-east-1")

	if localKey1 != localKey2 {
		t.Error("Identical local keys should be equal")
	}
	if localKey1 == regionalKey {
		t.Error("Local and regional keys should not be equal")
	}
}
