package metrics

import (
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg.ServerURL != _defaultServerURL || cfg.PushInterval != _defaultPushInterval ||
		cfg.Headers == nil || len(cfg.Headers) != 0 || cfg.Username != "" || cfg.Password != "" {
		t.Error("Config not properly initialized with defaults")
	}

	cfg.ServerURL = "https://custom.com/write"
	cfg.PushInterval = 10 * time.Second
	cfg.Username = "user"
	cfg.Password = "pass"
	cfg.Headers["Auth"] = "token"

	if cfg.ServerURL != "https://custom.com/write" || cfg.PushInterval != 10*time.Second ||
		cfg.Username != "user" || cfg.Password != "pass" || cfg.Headers["Auth"] != "token" {
		t.Error("Config customization failed")
	}
}

func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
	}{
		{"serverURL", _defaultServerURL, "http://localhost:9090/api/v1/write"},
		{"timeout", _defaultTimeout, 5 * time.Second},
		{"pushInterval", _defaultPushInterval, 5 * time.Second},
		{"metricPrefix", _defaultMetricPrefix, "updo_"},
	}

	for _, tt := range tests {
		if tt.value != tt.expected {
			t.Errorf("Constant %s: expected %v, got %v", tt.name, tt.expected, tt.value)
		}
	}
}
