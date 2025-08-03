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

	"github.com/Owloops/updo/aws"
	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/notifications"
	"github.com/Owloops/updo/stats"
	"github.com/Owloops/updo/utils"
)

const (
	_resultsChannelMultiplier = 2
	_signalChannelBuffer      = 1
)

const (
	requestFailedMsg   = "Request failed"
	assertionFailedMsg = "Assertion failed"
)

func getErrorMessage(result net.WebsiteCheckResult) string {
	if result.IsUp {
		return ""
	}
	switch {
	case result.StatusCode > 0:
		return fmt.Sprintf("Non-success status code: %d", result.StatusCode)
	case result.AssertText != "" && !result.AssertionPassed:
		return assertionFailedMsg
	default:
		return requestFailedMsg
	}
}

type TargetResult struct {
	Target   config.Target
	Result   net.WebsiteCheckResult
	Stats    stats.Stats
	Sequence int
	Region   string
}

type MonitoringOptions struct {
	Count   int
	Log     string
	Regions []string
	Profile string
}

func StartMultiTargetMonitoring(targets []config.Target, options MonitoringOptions) {
	if len(targets) == 0 {
		log.Fatal("No targets provided")
	}

	monitors := make(map[string]*stats.Monitor, len(targets))
	sequences := make(map[string]*int, len(targets))
	alertStates := make(map[string]*bool, len(targets))
	webhookAlertStates := make(map[string]*bool, len(targets))

	for _, target := range targets {
		monitor, err := stats.NewMonitor()
		if err != nil {
			log.Fatalf("Failed to initialize stats monitor for %s: %v", target.Name, err)
		}
		monitors[target.Name] = monitor
		var seq int
		var alert bool
		var webhookAlert bool
		sequences[target.Name] = &seq
		alertStates[target.Name] = &alert
		webhookAlertStates[target.Name] = &webhookAlert
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultsChan := make(chan TargetResult, len(targets)*_resultsChannelMultiplier)
	var wg sync.WaitGroup

	logMode := options.Log != ""

	outputManager := NewOutputManager(targets)
	if !logMode {
		outputManager.PrintHeader()
	}

	for _, target := range targets {
		wg.Add(1)
		go func(t config.Target) {
			defer wg.Done()
			monitorTarget(ctx, t, monitors[t.Name], sequences[t.Name], alertStates[t.Name], webhookAlertStates[t.Name], resultsChan, options)
		}(target)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	sigChan := make(chan os.Signal, _signalChannelBuffer)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	totalChecks := 0
	for {
		select {
		case result, ok := <-resultsChan:
			if !ok {
				return
			}

			totalChecks++
			if !logMode {
				outputManager.PrintResult(result)
			} else {
				utils.LogCheck(result.Result, result.Sequence, options.Log, result.Region)
				if !result.Result.IsUp {
					errorMsg := getErrorMessage(result.Result)
					utils.LogWarning(result.Target.URL, errorMsg, result.Region)
				}
			}

			if options.Count > 0 && totalChecks >= options.Count*len(targets) {
				outputManager.PrintFinalStatistics(monitors, targets, logMode)
				cancel()
				return
			}

		case <-sigChan:
			outputManager.PrintFinalStatistics(monitors, targets, logMode)
			cancel()
			return
		}
	}
}

func monitorTarget(ctx context.Context, target config.Target, monitor *stats.Monitor, sequence *int, alertSent *bool, webhookAlertSent *bool, resultsChan chan<- TargetResult, options MonitoringOptions) {
	ticker := time.NewTicker(target.GetRefreshInterval())
	defer ticker.Stop()

	attemptCount := 0

	makeRequest := func() {
		attemptCount++
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

		regions := target.Regions
		if len(regions) == 0 {
			regions = options.Regions
		}

		if len(regions) > 0 {
			lambdaResults := aws.InvokeMultiRegion(target.URL, netConfig, regions, options.Profile)
			for _, lambdaResult := range lambdaResults {
				if lambdaResult.Error != nil {
					utils.LogWarning(target.URL, fmt.Sprintf("Lambda invocation failed: %v", lambdaResult.Error), lambdaResult.Region)
					continue
				}
				monitor.AddResult(lambdaResult.Result)
				*sequence++

				if target.ReceiveAlert {
					notifications.HandleAlerts(lambdaResult.Result.IsUp, alertSent, target.Name, lambdaResult.Result.URL)
				}

				if target.WebhookURL != "" {
					errorMsg := getErrorMessage(lambdaResult.Result)
					if err := notifications.HandleWebhookAlert(target.WebhookURL, target.WebhookHeaders, lambdaResult.Result.IsUp, webhookAlertSent, target.Name, lambdaResult.Result.URL, lambdaResult.Result.ResponseTime, lambdaResult.Result.StatusCode, errorMsg); err != nil {
						log.Printf("[ERROR] %v", err)
					}
				}

				resultsChan <- TargetResult{
					Target:   target,
					Result:   lambdaResult.Result,
					Stats:    monitor.GetStats(),
					Sequence: *sequence,
					Region:   lambdaResult.Region,
				}
			}
		} else {
			result := net.CheckWebsite(target.URL, netConfig)
			monitor.AddResult(result)
			*sequence++

			if target.ReceiveAlert {
				notifications.HandleAlerts(result.IsUp, alertSent, target.Name, target.URL)
			}

			if target.WebhookURL != "" {
				errorMsg := getErrorMessage(result)
				if err := notifications.HandleWebhookAlert(target.WebhookURL, target.WebhookHeaders, result.IsUp, webhookAlertSent, target.Name, target.URL, result.ResponseTime, result.StatusCode, errorMsg); err != nil {
					log.Printf("[ERROR] %v", err)
				}
			}

			resultsChan <- TargetResult{
				Target:   target,
				Result:   result,
				Stats:    monitor.GetStats(),
				Sequence: *sequence,
				Region:   "",
			}
		}
	}

	makeRequest()

	if options.Count > 0 && attemptCount >= options.Count {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			makeRequest()

			if options.Count > 0 && attemptCount >= options.Count {
				return
			}
		}
	}
}
