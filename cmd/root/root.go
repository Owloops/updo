package root

import (
	"time"

	"github.com/spf13/cobra"
)

type Config struct {
	URL             string
	RefreshInterval time.Duration
	Timeout         time.Duration
	ShouldFail      bool
	FollowRedirects bool
	SkipSSL         bool
	AssertText      string
	ReceiveAlert    bool
	Simple          bool
	Count           int
	Headers         []string
	Method          string
	Body            string
	Log             string
}

var AppConfig Config

var RootCmd = &cobra.Command{
	Use:   "updo",
	Short: "A simple website monitoring tool",
	Long: `Updo is a lightweight, easy-to-use website monitoring tool that checks
website availability and response time. It provides both a terminal UI
and a simple text-based output mode.`,
	Example: `  updo monitor https://example.com
  updo --url https://example.com
  updo monitor -r 10 -t 5 https://example.com
  updo monitor --simple -c 10 https://example.com
  updo monitor --simple https://example.com
  updo monitor -a "Welcome" https://example.com
  updo --url https://example.com -H "Authorization: Bearer token123"
  updo monitor -X POST -H "Content-Type: application/json" https://api.example.com/endpoint
  updo monitor -X POST -d '{"name":"test"}' -H "Content-Type: application/json" https://api.example.com/data`,
}

func Execute() error {
	return RootCmd.Execute()
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&AppConfig.URL, "url", "u", "", "URL or IP address to monitor")
	RootCmd.PersistentFlags().IntP("refresh", "r", 5, "Refresh interval in seconds")
	RootCmd.PersistentFlags().IntP("timeout", "t", 10, "HTTP request timeout in seconds")
	RootCmd.PersistentFlags().BoolVarP(&AppConfig.ShouldFail, "should-fail", "f", false, "Invert success code range")
	RootCmd.PersistentFlags().BoolVarP(&AppConfig.FollowRedirects, "follow-redirects", "l", true, "Follow redirects")
	RootCmd.PersistentFlags().BoolVarP(&AppConfig.SkipSSL, "skip-ssl", "s", false, "Skip SSL certificate verification")
	RootCmd.PersistentFlags().StringVarP(&AppConfig.AssertText, "assert-text", "a", "", "Text to assert in the response body")
	RootCmd.PersistentFlags().BoolVarP(&AppConfig.ReceiveAlert, "receive-alert", "n", true, "Enable alert notifications")
	RootCmd.PersistentFlags().BoolVar(&AppConfig.Simple, "simple", false, "Use simple output instead of TUI")
	RootCmd.PersistentFlags().IntVarP(&AppConfig.Count, "count", "c", 0, "Number of checks to perform (0 = infinite)")
	RootCmd.PersistentFlags().StringArrayVarP(&AppConfig.Headers, "header", "H", nil, "HTTP header to send (can be used multiple times, format: 'Header-Name: value')")
	RootCmd.PersistentFlags().StringVarP(&AppConfig.Method, "request", "X", "GET", "HTTP request method to use")
	RootCmd.PersistentFlags().StringVarP(&AppConfig.Body, "data", "d", "", "HTTP request body data")

	var logEnabled bool
	RootCmd.PersistentFlags().BoolVar(&logEnabled, "log", false, "Output structured logs in JSON format (includes requests, responses, and metrics)")

	RootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		refresh, _ := cmd.Flags().GetInt("refresh")
		timeout, _ := cmd.Flags().GetInt("timeout")

		AppConfig.RefreshInterval = time.Duration(refresh) * time.Second
		AppConfig.Timeout = time.Duration(timeout) * time.Second

		logEnabled, _ := cmd.Flags().GetBool("log")
		if logEnabled {
			AppConfig.Log = "all"
		} else {
			AppConfig.Log = ""
		}
	}
}
