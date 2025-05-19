package simple

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
	"github.com/Owloops/updo/utils"
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
}

func StartMonitoring(config Config) {
	outputManager := NewOutputManager(config.URL)
	outputManager.PrintHeader()

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
					RefreshInterval: config.RefreshInterval,
				}

				result := net.CheckWebsite(config.URL, netConfig)
				monitor.AddResult(result)

				outputManager.PrintResult(result, monitor)

				if config.ReceiveAlert {
					utils.HandleAlerts(result.IsUp, &alertSent)
				}

				time.Sleep(config.RefreshInterval)
			}
		}

		close(doneChan)
	}()

	if config.Count > 0 {
		<-doneChan
	} else {
		<-sigChan
		close(doneChan)
	}

	stats := monitor.GetStats()
	outputManager.PrintStatistics(&stats)
}
