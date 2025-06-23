package monitor

import (
	"fmt"
	"os"

	"github.com/Owloops/updo/cmd/root"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/simple"
	"github.com/Owloops/updo/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var MonitorCmd = &cobra.Command{
	Use:   "monitor [url]",
	Short: "Monitor a URL and show its status",
	Long: `Monitor command checks a given URL at regular intervals and displays
its status, response time, and other metrics. It can operate in TUI mode
with a visual interface or in simple mode with text output.`,
	Example: `  updo monitor https://example.com
  updo monitor -r 10 -t 5 https://example.com
  updo monitor --simple -c 10 https://example.com
  updo monitor --simple --no-fancy https://example.com
  updo monitor -a "Welcome" https://example.com
  updo monitor -H "Authorization: Bearer token123" https://example.com`,
	Run: func(cmd *cobra.Command, args []string) {
		config := root.AppConfig

		if len(args) > 0 && config.URL == "" {
			config.URL = args[0]
		}

		if config.URL == "" {
			fmt.Println("Error: URL is required")
			fmt.Println("Use updo --help for usage information")
			os.Exit(1)
		}

		config.URL = net.AutoDetectProtocol(config.URL)

		useSimpleMode := config.Simple || !term.IsTerminal(int(os.Stdout.Fd()))

		if useSimpleMode {
			simpleConfig := simple.Config{
				URL:             config.URL,
				RefreshInterval: config.RefreshInterval,
				Timeout:         config.Timeout,
				ShouldFail:      config.ShouldFail,
				FollowRedirects: config.FollowRedirects,
				SkipSSL:         config.SkipSSL,
				AssertText:      config.AssertText,
				ReceiveAlert:    config.ReceiveAlert,
				Count:           config.Count,
				NoFancy:         config.NoFancy,
				Headers:         config.Headers,
			}

			simple.StartMonitoring(simpleConfig)
		} else {
			tuiConfig := tui.Config{
				URL:             config.URL,
				RefreshInterval: config.RefreshInterval,
				Timeout:         config.Timeout,
				ShouldFail:      config.ShouldFail,
				FollowRedirects: config.FollowRedirects,
				SkipSSL:         config.SkipSSL,
				AssertText:      config.AssertText,
				ReceiveAlert:    config.ReceiveAlert,
				Count:           config.Count,
				Headers:         config.Headers,
			}
			tui.StartMonitoring(tuiConfig)
		}
	},
}

func init() {
	// Add monitor-specific flags here if needed
}