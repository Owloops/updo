package simple

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Owloops/updo/aws"
	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/metrics"
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
	Count         int
	Log           string
	Regions       []string
	Profile       string
	PrometheusURL string
}

func StartMultiTargetMonitoring(targets []config.Target, options MonitoringOptions) {
	if len(targets) == 0 {
		log.Fatal("No targets provided")
	}

	keyRegistry := stats.NewTargetKeyRegistry(targets, options.Regions)
	allKeys := keyRegistry.GetAllKeys()

	monitors := make(map[string]*stats.Monitor, len(allKeys))
	sequences := make(map[string]*int, len(allKeys))
	alertStates := make(map[string]*bool, len(allKeys))
	webhookAlertStates := make(map[string]*bool, len(allKeys))

	for _, key := range allKeys {
		monitor, err := stats.NewMonitor()
		if err != nil {
			log.Fatalf("Failed to initialize stats monitor for %s: %v", key.String(), err)
		}
		keyStr := key.String()
		monitors[keyStr] = monitor
		var seq int
		var alert bool
		var webhookAlert bool
		sequences[keyStr] = &seq
		alertStates[keyStr] = &alert
		webhookAlertStates[keyStr] = &webhookAlert
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	prometheusURL := options.PrometheusURL
	if prometheusURL == "" {
		if updoURL := os.Getenv("UPDO_PROMETHEUS_RW_SERVER_URL"); updoURL != "" {
			prometheusURL = updoURL
		}
	}

	if prometheusURL != "" {
		metricsConfig := metrics.NewConfig()
		metricsConfig.ServerURL = prometheusURL

		if username := os.Getenv("UPDO_PROMETHEUS_USERNAME"); username != "" {
			metricsConfig.Username = username
		}
		if password := os.Getenv("UPDO_PROMETHEUS_PASSWORD"); password != "" {
			metricsConfig.Password = password
		}
		if bearerToken := os.Getenv("UPDO_PROMETHEUS_BEARER_TOKEN"); bearerToken != "" {
			metricsConfig.Headers["Authorization"] = "Bearer " + bearerToken
		}
		if authHeader := os.Getenv("UPDO_PROMETHEUS_AUTH_HEADER"); authHeader != "" {
			parts := strings.SplitN(authHeader, ":", 2)
			if len(parts) == 2 {
				metricsConfig.Headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
		if pushInterval := os.Getenv("UPDO_PROMETHEUS_PUSH_INTERVAL"); pushInterval != "" {
			if duration, err := time.ParseDuration(pushInterval); err == nil {
				metricsConfig.PushInterval = duration
			}
		}

		metrics.InitRemoteWrite(metricsConfig)
		defer metrics.StopRemoteWrite()
	}

	resultsChan := make(chan TargetResult, len(targets)*_resultsChannelMultiplier)
	var wg sync.WaitGroup

	logMode := options.Log != ""

	outputManager := NewOutputManager(targets)
	if !logMode {
		outputManager.PrintHeader()
	}

	for i, target := range targets {
		wg.Add(1)
		go func(t config.Target, index int) {
			defer wg.Done()
			monitorTargetSimple(ctx, t, index, monitors, sequences, alertStates, webhookAlertStates, resultsChan, options)
		}(target, i)
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

			if options.PrometheusURL != "" {
				metrics.RecordCheck(result.Target, result.Result, result.Region)

				if strings.HasPrefix(result.Target.URL, "https://") {
					if sslExpiry := net.GetSSLCertExpiry(result.Target.URL); sslExpiry >= 0 {
						metrics.RecordSSLExpiry(result.Target, sslExpiry)
					}
				}
			}

			if options.Count > 0 && totalChecks >= options.Count*len(targets) {
				outputManager.PrintFinalStatisticsWithKeys(monitors, keyRegistry, logMode)
				cancel()
				return
			}

		case <-sigChan:
			outputManager.PrintFinalStatisticsWithKeys(monitors, keyRegistry, logMode)
			cancel()
			return
		}
	}
}

func monitorTargetSimple(ctx context.Context, target config.Target, targetIndex int, monitors map[string]*stats.Monitor, sequences map[string]*int, alertStates map[string]*bool, webhookAlertStates map[string]*bool, resultsChan chan<- TargetResult, options MonitoringOptions) {
	ticker := time.NewTicker(target.GetRefreshInterval())
	defer ticker.Stop()

	attemptCount := 0

	makeRequest := func() {
		attemptCount++
		netConfig := net.NetworkConfig{
			Timeout:         target.GetTimeout(),
			ShouldFail:      target.ShouldFail,
			FollowRedirects: target.FollowRedirects,
			AcceptRedirects: target.AcceptRedirects,
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
				indexedName := fmt.Sprintf("%s#%d", target.Name, targetIndex)
				targetKey := stats.NewRegionTargetKey(indexedName, lambdaResult.Region, targetIndex)
				keyStr := targetKey.String()

				if monitor, exists := monitors[keyStr]; exists {
					if lambdaResult.Error != nil {
						utils.LogWarning(target.URL, fmt.Sprintf("Lambda invocation failed: %v", lambdaResult.Error), lambdaResult.Region)
						continue
					}

					monitor.AddResult(lambdaResult.Result)
					if sequence, exists := sequences[keyStr]; exists {
						*sequence++
					}

					if target.ReceiveAlert {
						if alertSent, exists := alertStates[keyStr]; exists {
							if err := notifications.HandleAlerts(lambdaResult.Result.IsUp, alertSent, target.Name, lambdaResult.Result.URL); err != nil {
								log.Printf("Alert notification failed: %v", err)
							}
						}
					}

					if target.WebhookURL != "" {
						errorMsg := getErrorMessage(lambdaResult.Result)
						if webhookAlertSent, exists := webhookAlertStates[keyStr]; exists {
							if err := notifications.HandleWebhookAlert(target.WebhookURL, target.WebhookHeaders, lambdaResult.Result.IsUp, webhookAlertSent, target.Name, lambdaResult.Result.URL, lambdaResult.Result.ResponseTime, lambdaResult.Result.StatusCode, errorMsg); err != nil {
								log.Printf("[ERROR] %v", err)
							}
						}
					}

					seq := 0
					if sequence, exists := sequences[keyStr]; exists {
						seq = *sequence
					}

					resultsChan <- TargetResult{
						Target:   target,
						Result:   lambdaResult.Result,
						Stats:    monitor.GetStats(),
						Sequence: seq,
						Region:   lambdaResult.Region,
					}
				}
			}
		} else {
			indexedName := fmt.Sprintf("%s#%d", target.Name, targetIndex)
			targetKey := stats.NewLocalTargetKey(indexedName, targetIndex)
			keyStr := targetKey.String()

			if monitor, exists := monitors[keyStr]; exists {
				result := net.CheckWebsite(target.URL, netConfig)
				monitor.AddResult(result)
				if sequence, exists := sequences[keyStr]; exists {
					*sequence++
				}

				if target.ReceiveAlert {
					if alertSent, exists := alertStates[keyStr]; exists {
						if err := notifications.HandleAlerts(result.IsUp, alertSent, target.Name, target.URL); err != nil {
							log.Printf("Alert notification failed: %v", err)
						}
					}
				}

				if target.WebhookURL != "" {
					errorMsg := getErrorMessage(result)
					if webhookAlertSent, exists := webhookAlertStates[keyStr]; exists {
						if err := notifications.HandleWebhookAlert(target.WebhookURL, target.WebhookHeaders, result.IsUp, webhookAlertSent, target.Name, target.URL, result.ResponseTime, result.StatusCode, errorMsg); err != nil {
							log.Printf("[ERROR] %v", err)
						}
					}
				}

				seq := 0
				if sequence, exists := sequences[keyStr]; exists {
					seq = *sequence
				}

				resultsChan <- TargetResult{
					Target:   target,
					Result:   result,
					Stats:    monitor.GetStats(),
					Sequence: seq,
					Region:   "",
				}
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
