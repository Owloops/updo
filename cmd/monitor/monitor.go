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
- Using --config flag with a TOML configuration file`,
	Example: `  updo monitor https://example.com
  updo monitor https://example.com https://google.com
  updo monitor --config updo.toml
  updo monitor -r 10 -t 5 https://example.com
  updo monitor --simple -c 10 https://example.com
  updo monitor -a "Welcome" https://example.com
  updo monitor -H "Authorization: Bearer token123" https://example.com`,
	Run: func(cmd *cobra.Command, args []string) {
		appConfig := root.AppConfig
		regions, _ := cmd.Flags().GetStringSlice("regions")
		profile, _ := cmd.Flags().GetString("profile")

		var targets []config.Target

		if appConfig.ConfigFile != "" {
			cfg, err := config.LoadConfig(appConfig.ConfigFile)
			if err != nil {
				fmt.Printf("Error loading config file: %v\n", err)
				os.Exit(1)
			}
			targets = cfg.FilterTargets(appConfig.Only, appConfig.Skip)
			if appConfig.Count == 0 && cfg.Global.Count > 0 {
				appConfig.Count = cfg.Global.Count
			}
		} else {
			var urls []string

			if appConfig.URL != "" {
				urls = append(urls, appConfig.URL)
			}
			if len(args) > 0 {
				urls = append(urls, args...)
			}

			if len(urls) == 0 {
				fmt.Println("Error: At least one URL is required")
				fmt.Println("Use updo monitor --help for usage information")
				os.Exit(1)
			}

			targets = make([]config.Target, 0, len(urls))
			for i, url := range urls {
				target := config.Target{
					URL:             net.AutoDetectProtocol(url),
					Name:            fmt.Sprintf("Target-%d", i+1),
					RefreshInterval: int(appConfig.RefreshInterval.Seconds()),
					Timeout:         int(appConfig.Timeout.Seconds()),
					ShouldFail:      appConfig.ShouldFail,
					FollowRedirects: appConfig.FollowRedirects,
					AcceptRedirects: appConfig.AcceptRedirects,
					SkipSSL:         appConfig.SkipSSL,
					AssertText:      appConfig.AssertText,
					ReceiveAlert:    appConfig.ReceiveAlert,
					Headers:         appConfig.Headers,
					Method:          appConfig.Method,
					Body:            appConfig.Body,
					WebhookURL:      appConfig.WebhookURL,
					WebhookHeaders:  appConfig.WebhookHeaders,
				}
				targets = append(targets, target)
			}
		}

		if len(targets) == 0 {
			fmt.Println("Error: No targets to monitor after filtering")
			fmt.Println("Check your --only/--skip flags or config file settings")
			os.Exit(1)
		}

		useSimpleMode := appConfig.Simple || !term.IsTerminal(int(os.Stdout.Fd()))

		if useSimpleMode {
			options := simple.MonitoringOptions{
				Count:         appConfig.Count,
				Log:           appConfig.Log,
				Regions:       regions,
				Profile:       profile,
				PrometheusURL: appConfig.PrometheusURL,
			}
			simple.StartMultiTargetMonitoring(targets, options)
		} else {
			options := tui.Options{
				Count:         appConfig.Count,
				Log:           appConfig.Log,
				Regions:       regions,
				Profile:       profile,
				PrometheusURL: appConfig.PrometheusURL,
			}
			tui.StartMonitoring(targets, options)
		}
	},
}

func init() {
	MonitorCmd.Flags().StringSlice("regions", nil, "AWS regions to invoke Lambda functions for multi-region checks")
	MonitorCmd.Flags().String("profile", "", "AWS profile to use for Lambda invocations")
}
