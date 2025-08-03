package config

import (
	"time"

	"github.com/spf13/viper"
)

const (
	_defaultRefreshInterval = 5
	_defaultTimeout         = 10
	_defaultMethod          = "GET"
)

type Target struct {
	URL             string   `mapstructure:"url"`
	Name            string   `mapstructure:"name"`
	RefreshInterval int      `mapstructure:"refresh_interval"`
	Timeout         int      `mapstructure:"timeout"`
	ShouldFail      bool     `mapstructure:"should_fail"`
	FollowRedirects bool     `mapstructure:"follow_redirects"`
	SkipSSL         bool     `mapstructure:"skip_ssl"`
	AssertText      string   `mapstructure:"assert_text"`
	ReceiveAlert    bool     `mapstructure:"receive_alert"`
	Headers         []string `mapstructure:"headers"`
	Method          string   `mapstructure:"method"`
	Body            string   `mapstructure:"body"`
	WebhookURL      string   `mapstructure:"webhook_url"`
	WebhookHeaders  []string `mapstructure:"webhook_headers"`
	Regions         []string `mapstructure:"regions"`
}

type Global struct {
	RefreshInterval int      `mapstructure:"refresh_interval"`
	Timeout         int      `mapstructure:"timeout"`
	ShouldFail      bool     `mapstructure:"should_fail"`
	FollowRedirects bool     `mapstructure:"follow_redirects"`
	SkipSSL         bool     `mapstructure:"skip_ssl"`
	ReceiveAlert    bool     `mapstructure:"receive_alert"`
	Count           int      `mapstructure:"count"`
	Simple          bool     `mapstructure:"simple"`
	Log             bool     `mapstructure:"log"`
	Only            []string `mapstructure:"only"`
	Skip            []string `mapstructure:"skip"`
	WebhookURL      string   `mapstructure:"webhook_url"`
	WebhookHeaders  []string `mapstructure:"webhook_headers"`
	Regions         []string `mapstructure:"regions"`
}

type Config struct {
	Global  Global   `mapstructure:"global"`
	Targets []Target `mapstructure:"targets"`
}

func LoadConfig(configFile string) (*Config, error) {
	viper.SetConfigFile(configFile)
	viper.SetConfigType("toml")

	viper.SetDefault("global.refresh_interval", _defaultRefreshInterval)
	viper.SetDefault("global.timeout", _defaultTimeout)
	viper.SetDefault("global.follow_redirects", true)
	viper.SetDefault("global.receive_alert", true)
	viper.SetDefault("global.count", 0)
	viper.SetDefault("global.method", _defaultMethod)

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	for i := range config.Targets {
		target := &config.Targets[i]
		if target.RefreshInterval == 0 {
			target.RefreshInterval = config.Global.RefreshInterval
		}
		if target.Timeout == 0 {
			target.Timeout = config.Global.Timeout
		}
		if target.Method == "" {
			target.Method = _defaultMethod
		}
		if !target.FollowRedirects && config.Global.FollowRedirects {
			target.FollowRedirects = config.Global.FollowRedirects
		}
		if !target.ReceiveAlert && config.Global.ReceiveAlert {
			target.ReceiveAlert = config.Global.ReceiveAlert
		}
		if target.WebhookURL == "" && config.Global.WebhookURL != "" {
			target.WebhookURL = config.Global.WebhookURL
		}
		if len(target.WebhookHeaders) == 0 && len(config.Global.WebhookHeaders) > 0 {
			target.WebhookHeaders = config.Global.WebhookHeaders
		}
		if len(target.Regions) == 0 && len(config.Global.Regions) > 0 {
			target.Regions = config.Global.Regions
		}
	}

	return &config, nil
}

func (t *Target) GetRefreshInterval() time.Duration {
	return time.Duration(t.RefreshInterval) * time.Second
}

func (t *Target) GetTimeout() time.Duration {
	return time.Duration(t.Timeout) * time.Second
}

func (g *Global) GetRefreshInterval() time.Duration {
	return time.Duration(g.RefreshInterval) * time.Second
}

func (g *Global) GetTimeout() time.Duration {
	return time.Duration(g.Timeout) * time.Second
}

func (c *Config) FilterTargets(onlyFlags, skipFlags []string) []Target {
	only := onlyFlags
	skip := skipFlags

	if len(only) == 0 {
		only = c.Global.Only
	}
	if len(skip) == 0 {
		skip = c.Global.Skip
	}

	if len(only) == 0 && len(skip) == 0 {
		return c.Targets
	}

	var filtered []Target

	for _, target := range c.Targets {
		targetName := getTargetName(target)

		if len(only) > 0 && !containsTarget(only, targetName, target.URL) {
			continue
		}

		if len(skip) > 0 && containsTarget(skip, targetName, target.URL) {
			continue
		}

		filtered = append(filtered, target)
	}

	return filtered
}

func getTargetName(target Target) string {
	if target.Name != "" {
		return target.Name
	}
	return target.URL
}

func containsTarget(list []string, targetName, targetURL string) bool {
	for _, name := range list {
		if name == targetName || name == targetURL {
			return true
		}
	}
	return false
}
