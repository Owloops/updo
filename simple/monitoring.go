package simple

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
	"github.com/Owloops/updo/utils"
	"golang.org/x/term"
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
	Count           int
	NoFancy         bool
	Headers         []string
	Method          string
	Body            string
	Log             string
}

func StartMonitoring(config Config) {
	logMode := config.Log != ""

	isTerminal := term.IsTerminal(int(os.Stdout.Fd()))

	if isTerminal && !config.NoFancy {
		StartBubbleTeaMonitoring(config)
		return
	}

	outputManager := NewOutputManager(config.URL)

	if !logMode {
		outputManager.PrintHeader()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	alertSent := false

	monitor, err := stats.NewMonitor()
	if err != nil {
		log.Fatalf("Failed to initialize stats monitor: %v", err)
	}

	doneChan := make(chan bool)

	go func() {
		for monitor.ChecksCount < config.Count || config.Count == 0 {
			select {
			case <-doneChan:
				return
			default:
				netConfig := net.NetworkConfig{
					Timeout:         config.Timeout,
					ShouldFail:      config.ShouldFail,
					FollowRedirects: config.FollowRedirects,
					SkipSSL:         config.SkipSSL,
					AssertText:      config.AssertText,
					Headers:         config.Headers,
					Method:          config.Method,
					Body:            config.Body,
				}

				result := net.CheckWebsite(config.URL, netConfig)
				monitor.AddResult(result)

				if !logMode {
					outputManager.PrintResult(result, monitor)
				} else {
					utils.LogCheck(result, monitor.ChecksCount-1, config.Log)

					if !result.IsUp {
						errorMsg := "Request failed"
						if result.StatusCode > 0 {
							errorMsg = fmt.Sprintf("Non-success status code: %d", result.StatusCode)
						} else if result.AssertText != "" && !result.AssertionPassed {
							errorMsg = "Assertion failed"
						}
						utils.LogWarning(config.URL, errorMsg)
					}
				}

				if config.ReceiveAlert {
					utils.HandleAlerts(result.IsUp, &alertSent)
				}

				time.Sleep(config.RefreshInterval)
			}
		}

		close(doneChan)
	}()

	select {
	case <-doneChan:
	case <-sigChan:
		close(doneChan)
	}

	stats := monitor.GetStats()

	if !logMode {
		outputManager.PrintStatistics(&stats)
	} else {
		utils.LogMetrics(&stats, config.URL)
	}
}
