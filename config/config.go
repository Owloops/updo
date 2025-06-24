package config

import (
	"time"

	"github.com/spf13/viper"
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
}

type Global struct {
	RefreshInterval int  `mapstructure:"refresh_interval"`
	Timeout         int  `mapstructure:"timeout"`
	ShouldFail      bool `mapstructure:"should_fail"`
	FollowRedirects bool `mapstructure:"follow_redirects"`
	SkipSSL         bool `mapstructure:"skip_ssl"`
	ReceiveAlert    bool `mapstructure:"receive_alert"`
	Count           int  `mapstructure:"count"`
	Simple          bool `mapstructure:"simple"`
	Log             bool `mapstructure:"log"`
}

type Config struct {
	Global  Global   `mapstructure:"global"`
	Targets []Target `mapstructure:"targets"`
}

func LoadConfig(configFile string) (*Config, error) {
	viper.SetConfigFile(configFile)
	viper.SetConfigType("toml")

	viper.SetDefault("global.refresh_interval", 5)
	viper.SetDefault("global.timeout", 10)
	viper.SetDefault("global.follow_redirects", true)
	viper.SetDefault("global.receive_alert", true)
	viper.SetDefault("global.count", 0)
	viper.SetDefault("global.method", "GET")

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
			target.Method = "GET"
		}
		if !target.FollowRedirects && config.Global.FollowRedirects {
			target.FollowRedirects = config.Global.FollowRedirects
		}
		if !target.ReceiveAlert && config.Global.ReceiveAlert {
			target.ReceiveAlert = config.Global.ReceiveAlert
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
