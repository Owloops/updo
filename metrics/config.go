package metrics

import (
	"time"
)

const (
	_defaultServerURL    = "http://localhost:9090/api/v1/write"
	_defaultTimeout      = 5 * time.Second
	_defaultPushInterval = 5 * time.Second
	_defaultMetricPrefix = "updo_"
)

type Config struct {
	ServerURL    string
	Headers      map[string]string
	PushInterval time.Duration
	Username     string
	Password     string
}

func NewConfig() Config {
	return Config{
		ServerURL:    _defaultServerURL,
		PushInterval: _defaultPushInterval,
		Headers:      make(map[string]string),
	}
}
