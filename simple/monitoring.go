package simple

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
	"github.com/Owloops/updo/utils"
)

type TargetResult struct {
	Target   config.Target
	Result   net.WebsiteCheckResult
	Stats    stats.Stats
	Sequence int
}

type MonitoringOptions struct {
	Count int
	Log   string
}

func StartMultiTargetMonitoring(targets []config.Target, options MonitoringOptions) {
	if len(targets) == 0 {
		log.Fatal("No targets provided")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultsChan := make(chan TargetResult, len(targets)*2)
	var wg sync.WaitGroup

	monitors := make(map[string]*stats.Monitor)
	sequences := make(map[string]*int)
	alertStates := make(map[string]*bool)

	for _, target := range targets {
		monitor, err := stats.NewMonitor()
		if err != nil {
			cancel()
			log.Fatalf("Failed to initialize stats monitor for %s: %v", target.Name, err)
		}
		monitors[target.Name] = monitor
		seq := 0
		alert := false
		sequences[target.Name] = &seq
		alertStates[target.Name] = &alert
	}

	logMode := options.Log != ""

	outputManager := NewOutputManager(targets)
	if !logMode {
		outputManager.PrintHeader()
	}

	for _, target := range targets {
		wg.Add(1)
		go func(t config.Target) {
			defer wg.Done()
			monitorTarget(ctx, t, monitors[t.Name], sequences[t.Name], alertStates[t.Name], resultsChan, options)
		}(target)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	totalChecks := 0
	for {
		select {
		case result, ok := <-resultsChan:
			if !ok {
				if !logMode {
					outputManager.PrintStatistics(monitors)
				} else {
					for _, target := range targets {
						stats := monitors[target.Name].GetStats()
						utils.LogMetrics(&stats, target.URL)
					}
				}
				return
			}

			totalChecks++
			if !logMode {
				outputManager.PrintResult(result)
			} else {
				utils.LogCheck(result.Result, result.Sequence, options.Log)
				if !result.Result.IsUp {
					errorMsg := "Request failed"
					if result.Result.StatusCode > 0 {
						errorMsg = fmt.Sprintf("Non-success status code: %d", result.Result.StatusCode)
					} else if result.Result.AssertText != "" && !result.Result.AssertionPassed {
						errorMsg = "Assertion failed"
					}
					utils.LogWarning(result.Target.URL, errorMsg)
				}
			}

			if options.Count > 0 && totalChecks >= options.Count*len(targets) {
				cancel()
			}

		case <-sigChan:
			cancel()
		}
	}
}

func monitorTarget(ctx context.Context, target config.Target, monitor *stats.Monitor, sequence *int, alertSent *bool, resultsChan chan<- TargetResult, options MonitoringOptions) {
	ticker := time.NewTicker(target.GetRefreshInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			netConfig := net.NetworkConfig{
				Timeout:         target.GetTimeout(),
				ShouldFail:      target.ShouldFail,
				FollowRedirects: target.FollowRedirects,
				SkipSSL:         target.SkipSSL,
				AssertText:      target.AssertText,
				Headers:         target.Headers,
				Method:          target.Method,
				Body:            target.Body,
			}

			result := net.CheckWebsite(target.URL, netConfig)
			monitor.AddResult(result)
			*sequence++

			if target.ReceiveAlert {
				utils.HandleAlerts(result.IsUp, alertSent)
			}

			stats := monitor.GetStats()
			resultsChan <- TargetResult{
				Target:   target,
				Result:   result,
				Stats:    stats,
				Sequence: *sequence - 1,
			}

			if options.Count > 0 && monitor.ChecksCount >= options.Count {
				return
			}
		}
	}
}
