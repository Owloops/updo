package monitor

import (
	"fmt"
	"os"

	"github.com/Owloops/updo/cmd/root"
	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/simple"
	"github.com/Owloops/updo/stats"
	"github.com/Owloops/updo/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	_exitFailure = 1
	_exitUsage   = 2
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
				os.Exit(_exitUsage)
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
				os.Exit(_exitUsage)
			}

			targets = make([]config.Target, 0, len(urls))
			for i, url := range urls {
				target := config.Target{
					URL:             net.AutoDetectProtocol(url),
					Name:            fmt.Sprintf("Target-%d", i+1),
					RefreshInterval: int(appConfig.RefreshInterval.Seconds()),
					Timeout:         int(appConfig.Timeout.Seconds()),
					ShouldFail:      appConfig.ShouldFail,
					FollowRedirects: &appConfig.FollowRedirects,
					AcceptRedirects: &appConfig.AcceptRedirects,
					SkipSSL:         &appConfig.SkipSSL,
					AssertText:      appConfig.AssertText,
					ReceiveAlert:    &appConfig.ReceiveAlert,
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
			os.Exit(_exitUsage)
		}

		useSimpleMode := appConfig.Simple || !term.IsTerminal(int(os.Stdout.Fd()))

		var summary stats.Summary
		if useSimpleMode {
			summary = simple.StartMultiTargetMonitoring(targets, simpleOptions(appConfig, regions, profile))
			os.Exit(exitCode(summary))
		}

		var err error
		summary, err = tui.StartMonitoring(targets, tui.Options{
			Count:         appConfig.Count,
			Log:           appConfig.Log,
			Regions:       regions,
			Profile:       profile,
			PrometheusURL: appConfig.PrometheusURL,
			IPVersion:     ipVersion(appConfig),
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v, falling back to simple output\n", err)
			summary = simple.StartMultiTargetMonitoring(targets, simpleOptions(appConfig, regions, profile))
		}

		os.Exit(exitCode(summary))
	},
}

func simpleOptions(cfg root.Config, regions []string, profile string) simple.MonitoringOptions {
	return simple.MonitoringOptions{
		Count:         cfg.Count,
		Log:           cfg.Log,
		Regions:       regions,
		Profile:       profile,
		PrometheusURL: cfg.PrometheusURL,
		IPVersion:     ipVersion(cfg),
	}
}

// exitCode reports 1 when a completed run left a target down, and 0 otherwise,
// including runs the user interrupted.
func exitCode(summary stats.Summary) int {
	if summary.Interrupted || summary.AllUp {
		return 0
	}
	return _exitFailure
}

func ipVersion(cfg root.Config) net.IPVersion {
	switch {
	case cfg.IPv4:
		return net.IPv4Only
	case cfg.IPv6:
		return net.IPv6Only
	}
	return net.IPAny
}

func init() {
	MonitorCmd.Flags().StringSlice("regions", nil, "AWS regions to invoke Lambda functions for multi-region checks")
	MonitorCmd.Flags().String("profile", "", "AWS profile to use for Lambda invocations")
}
