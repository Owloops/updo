package monitor

import (
	"fmt"
	"os"

	"github.com/Owloops/updo/cmd/root"
	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/simple"
	"github.com/Owloops/updo/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var MonitorCmd = &cobra.Command{
	Use:   "monitor [url...]",
	Short: "Monitor one or more URLs and show their status",
	Long: `Monitor command checks given URLs at regular intervals and displays
their status, response time, and other metrics. It can operate in TUI mode
with a visual interface or in simple mode with text output.

You can monitor multiple targets by:
- Providing multiple URLs as arguments
- Using --urls flag with comma-separated URLs  
- Using --config flag with a TOML configuration file`,
	Example: `  updo monitor https://example.com
  updo monitor https://example.com https://google.com
  updo monitor --urls="https://example.com,https://google.com"
  updo monitor --config updo.toml
  updo monitor -r 10 -t 5 https://example.com
  updo monitor --simple -c 10 https://example.com
  updo monitor -a "Welcome" https://example.com
  updo monitor -H "Authorization: Bearer token123" https://example.com`,
	Run: func(cmd *cobra.Command, args []string) {
		appConfig := root.AppConfig

		var targets []config.Target

		if appConfig.ConfigFile != "" {
			cfg, err := config.LoadConfig(appConfig.ConfigFile)
			if err != nil {
				fmt.Printf("Error loading config file: %v\n", err)
				os.Exit(1)
			}
			targets = cfg.Targets
			if appConfig.Count == 0 && cfg.Global.Count > 0 {
				appConfig.Count = cfg.Global.Count
			}
		} else {
			var urls []string

			if appConfig.URL != "" {
				urls = append(urls, appConfig.URL)
			}
			if len(appConfig.URLs) > 0 {
				urls = append(urls, appConfig.URLs...)
			}
			if len(args) > 0 {
				urls = append(urls, args...)
			}

			if len(urls) == 0 {
				fmt.Println("Error: At least one URL is required")
				fmt.Println("Use updo monitor --help for usage information")
				os.Exit(1)
			}

			targets = make([]config.Target, len(urls))
			for i, url := range urls {
				targets[i] = config.Target{
					URL:             net.AutoDetectProtocol(url),
					Name:            fmt.Sprintf("Target-%d", i+1),
					RefreshInterval: int(appConfig.RefreshInterval.Seconds()),
					Timeout:         int(appConfig.Timeout.Seconds()),
					ShouldFail:      appConfig.ShouldFail,
					FollowRedirects: appConfig.FollowRedirects,
					SkipSSL:         appConfig.SkipSSL,
					AssertText:      appConfig.AssertText,
					ReceiveAlert:    appConfig.ReceiveAlert,
					Headers:         appConfig.Headers,
					Method:          appConfig.Method,
					Body:            appConfig.Body,
				}
			}
		}

		useSimpleMode := appConfig.Simple || !term.IsTerminal(int(os.Stdout.Fd()))

		if useSimpleMode {
			options := simple.MonitoringOptions{
				Count: appConfig.Count,
				Log:   appConfig.Log,
			}
			simple.StartMultiTargetMonitoring(targets, options)
		} else {
			if len(targets) > 1 {
				fmt.Println("TUI mode with multiple targets not yet implemented. Using simple mode.")
				options := simple.MonitoringOptions{
					Count: appConfig.Count,
					Log:   appConfig.Log,
				}
				simple.StartMultiTargetMonitoring(targets, options)
			} else {
				target := targets[0]
				tuiConfig := tui.Config{
					URL:             target.URL,
					RefreshInterval: target.GetRefreshInterval(),
					Timeout:         target.GetTimeout(),
					ShouldFail:      target.ShouldFail,
					FollowRedirects: target.FollowRedirects,
					SkipSSL:         target.SkipSSL,
					AssertText:      target.AssertText,
					ReceiveAlert:    target.ReceiveAlert,
					Count:           appConfig.Count,
					Headers:         target.Headers,
					Method:          target.Method,
					Body:            target.Body,
					Log:             appConfig.Log,
				}
				tui.StartMonitoring(tuiConfig)
			}
		}
	},
}

func init() {
	// Add monitor-specific flags here if needed
}
