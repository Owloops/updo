package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Owloops/updo/aws"
	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/metrics"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/notifications"
	"github.com/Owloops/updo/stats"
	ui "github.com/gizak/termui/v3"
)

type TargetData struct {
	Target       config.Target
	Result       net.WebsiteCheckResult
	Stats        stats.Stats
	TargetKey    stats.TargetKey
	WebhookError error
	LambdaError  error
	AlertError   error
}

type Options struct {
	Count         int
	Log           string
	Regions       []string
	Profile       string
	PrometheusURL string
}

func StartMonitoring(targets []config.Target, options Options) {
	if len(targets) == 0 {
		panic("No targets provided")
	}

	if err := ui.Init(); err != nil {
		panic(err)
	}
	defer ui.Close()

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

	keyRegistry := stats.NewTargetKeyRegistry(targets, options.Regions)
	allKeys := keyRegistry.GetAllKeys()

	monitors := make(map[string]*stats.Monitor, len(allKeys))
	sequences := make(map[string]*int, len(allKeys))
	alertStates := make(map[string]*bool, len(allKeys))
	webhookAlertStates := make(map[string]*bool, len(allKeys))

	for _, key := range allKeys {
		monitor, err := stats.NewMonitor()
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize stats monitor for %s: %v", key.String(), err))
		}
		monitors[key.String()] = monitor
		seq := 0
		alert := false
		webhookAlert := false
		sequences[key.String()] = &seq
		alertStates[key.String()] = &alert
		webhookAlertStates[key.String()] = &webhookAlert
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataChannel := make(chan TargetData, len(targets)*_dataChannelMultiplier)
	var wg sync.WaitGroup

	for i, target := range targets {
		wg.Add(1)
		go func(t config.Target, index int) {
			defer wg.Done()
			monitorTargetTUI(ctx, t, index, monitors, sequences, alertStates, webhookAlertStates, dataChannel, options)
		}(target, i)
	}

	go func() {
		wg.Wait()
		close(dataChannel)
	}()

	manager := NewManager(targets, options)
	width, height := ui.TerminalDimensions()
	manager.InitializeLayout(width, height)

	uiRefreshTicker := time.NewTicker(1 * time.Second)
	defer uiRefreshTicker.Stop()

	uiEvents := ui.PollEvents()

	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q":
				if manager.listWidget != nil && manager.listWidget.IsSearchMode() {
					manager.listWidget.UpdateSearch("q")
					ui.Render(manager.grid)
				} else {
					cancel()
					return
				}
			case "<C-c>":
				cancel()
				return
			case "<Resize>":
				if payload, ok := e.Payload.(ui.Resize); ok {
					width, height = payload.Width, payload.Height
					manager.Resize(width, height)
					ui.Render(manager.grid)
				}
			case "<Down>":
				if !manager.isSingle {
					if manager.IsFocusedOnLogs() {
						manager.NavigateLogs(1)
					} else {
						manager.NavigateTargetKeys(1, monitors)
					}
					ui.Render(manager.grid)
				}
			case "<Up>":
				if !manager.isSingle {
					if manager.IsFocusedOnLogs() {
						manager.NavigateLogs(-1)
					} else {
						manager.NavigateTargetKeys(-1, monitors)
					}
					ui.Render(manager.grid)
				}
			case "<Enter>":
				if manager.IsFocusedOnLogs() && manager.detailsManager.LogsWidget != nil {
					func() {
						defer func() {
							if r := recover(); r != nil {
								_ = r
							}
						}()
						manager.detailsManager.LogsWidget.ToggleExpand()
					}()
					ui.Render(manager.grid)
				} else if manager.listWidget != nil && manager.listWidget.IsHeaderAtIndex(manager.listWidget.SelectedRow) {
					groupID := manager.listWidget.GetGroupAtIndex(manager.listWidget.SelectedRow)
					if groupID != "" {
						manager.preserveHeaderSelection = groupID
						manager.listWidget.ToggleGroupCollapse(groupID)
						manager.updateTargetList()
						manager.preserveHeaderSelection = ""
						ui.Render(manager.grid)
					}
				}
			case "l":
				if manager.listWidget != nil && manager.listWidget.IsSearchMode() {
					manager.listWidget.UpdateSearch("l")
				} else {
					manager.ToggleLogsVisibility()
				}
				ui.Render(manager.grid)
			case "/":
				if len(targets) > 1 && manager.listWidget != nil {
					manager.listWidget.ToggleSearch()
					if manager.listWidget.IsSearchMode() && manager.listWidget.OnSearchChange != nil {
						indices := manager.listWidget.GetFilteredIndices()
						manager.listWidget.OnSearchChange(manager.listWidget.GetQuery(), indices)
					}
					ui.Render(manager.grid)
				}
			case "<Escape>":
				if manager.listWidget != nil && manager.listWidget.IsSearchMode() {
					manager.listWidget.ToggleSearch()
					if manager.listWidget.OnSearchChange != nil {
						manager.listWidget.OnSearchChange(manager.listWidget.GetQuery(), manager.listWidget.GetFilteredIndices())
					}
					ui.Render(manager.grid)
				}
			case _backspaceKey, _ctrlBackspace, "<Space>":
				if manager.listWidget != nil && manager.listWidget.IsSearchMode() {
					manager.listWidget.UpdateSearch(e.ID)
					ui.Render(manager.grid)
				}
			case "<Tab>":
				if manager.listWidget != nil && !manager.listWidget.IsSearchMode() {
					manager.listWidget.ToggleAllGroups()
					manager.updateTargetList()
					ui.Render(manager.grid)
				}
			default:
				if manager.listWidget != nil && manager.listWidget.IsSearchMode() && len(e.ID) == 1 {
					manager.listWidget.UpdateSearch(e.ID)
					ui.Render(manager.grid)
				}
			}

		case data, ok := <-dataChannel:
			if !ok {
				return
			}
			manager.UpdateTarget(data)

			if options.PrometheusURL != "" {
				region := ""
				if !data.TargetKey.IsLocal {
					region = data.TargetKey.Region
				}
				metrics.RecordCheck(data.Target, data.Result, region)

				if strings.HasPrefix(data.Target.URL, "https://") {
					go func(target config.Target) {
						if sslExpiry := net.GetSSLCertExpiry(target.URL); sslExpiry >= 0 {
							metrics.RecordSSLExpiry(target, sslExpiry)
						}
					}(data.Target)
				}
			}

		case <-uiRefreshTicker.C:
			manager.RefreshStats(monitors)
		}
	}
}

func monitorTargetTUI(ctx context.Context, target config.Target, targetIndex int, monitors map[string]*stats.Monitor, sequences map[string]*int, alertStates map[string]*bool, webhookAlertStates map[string]*bool, dataChannel chan<- TargetData, options Options) {
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
				if lambdaResult.Error != nil {
					errorResult := net.WebsiteCheckResult{
						URL:           target.URL,
						IsUp:          false,
						StatusCode:    0,
						LastCheckTime: time.Now(),
					}

					indexedName := fmt.Sprintf("%s#%d", target.Name, targetIndex)
					targetKey := stats.NewRegionTargetKey(indexedName, lambdaResult.Region, targetIndex)
					dataChannel <- TargetData{
						Target:      target,
						Result:      errorResult,
						Stats:       stats.Stats{},
						TargetKey:   targetKey,
						LambdaError: lambdaResult.Error,
					}
					continue
				}

				indexedName := fmt.Sprintf("%s#%d", target.Name, targetIndex)
				targetKey := stats.NewRegionTargetKey(indexedName, lambdaResult.Region, targetIndex)
				targetKeyStr := targetKey.String()

				if monitor, exists := monitors[targetKeyStr]; exists {
					monitor.AddResult(lambdaResult.Result)
					if sequence, exists := sequences[targetKeyStr]; exists {
						*sequence++
					}

					if target.ReceiveAlert {
						if alertSent, exists := alertStates[targetKeyStr]; exists {
							if err := notifications.HandleAlerts(lambdaResult.Result.IsUp, alertSent, target.Name, lambdaResult.Result.URL); err != nil {
								dataChannel <- TargetData{
									Target:     target,
									Result:     lambdaResult.Result,
									Stats:      monitor.GetStats(),
									TargetKey:  targetKey,
									AlertError: err,
								}
							}
						}
					}

					if target.WebhookURL != "" {
						errorMsg := ""
						if !lambdaResult.Result.IsUp {
							switch {
							case lambdaResult.Result.StatusCode > 0:
								errorMsg = fmt.Sprintf("Non-success status code: %d", lambdaResult.Result.StatusCode)
							case lambdaResult.Result.AssertText != "" && !lambdaResult.Result.AssertionPassed:
								errorMsg = "Assertion failed"
							default:
								errorMsg = "Request failed"
							}
						}
						if webhookAlertSent, exists := webhookAlertStates[targetKeyStr]; exists {
							if err := notifications.HandleWebhookAlert(target.WebhookURL, target.WebhookHeaders, lambdaResult.Result.IsUp, webhookAlertSent, target.Name, lambdaResult.Result.URL, lambdaResult.Result.ResponseTime, lambdaResult.Result.StatusCode, errorMsg); err != nil {
								dataChannel <- TargetData{
									Target:       target,
									Result:       lambdaResult.Result,
									Stats:        stats.Stats{},
									TargetKey:    targetKey,
									WebhookError: err,
								}
							}
						}
					}

					stats := monitor.GetStats()
					dataChannel <- TargetData{
						Target:    target,
						Result:    lambdaResult.Result,
						Stats:     stats,
						TargetKey: targetKey,
					}
				}
			}
		} else {
			result := net.CheckWebsite(target.URL, netConfig)
			indexedName := fmt.Sprintf("%s#%d", target.Name, targetIndex)
			targetKey := stats.NewLocalTargetKey(indexedName, targetIndex)
			targetKeyStr := targetKey.String()

			if monitor, exists := monitors[targetKeyStr]; exists {
				monitor.AddResult(result)
				if sequence, exists := sequences[targetKeyStr]; exists {
					*sequence++
				}

				if target.ReceiveAlert {
					if alertSent, exists := alertStates[targetKeyStr]; exists {
						if err := notifications.HandleAlerts(result.IsUp, alertSent, target.Name, target.URL); err != nil {
							stats := monitor.GetStats()
							dataChannel <- TargetData{
								Target:     target,
								Result:     result,
								Stats:      stats,
								TargetKey:  targetKey,
								AlertError: err,
							}
						}
					}
				}

				if target.WebhookURL != "" {
					errorMsg := ""
					if !result.IsUp {
						errorMsg = fmt.Sprintf("Status code: %d", result.StatusCode)
					}
					if webhookAlertSent, exists := webhookAlertStates[targetKeyStr]; exists {
						if err := notifications.HandleWebhookAlert(
							target.WebhookURL,
							target.WebhookHeaders,
							result.IsUp,
							webhookAlertSent,
							target.Name,
							target.URL,
							result.ResponseTime,
							result.StatusCode,
							errorMsg,
						); err != nil {
							dataChannel <- TargetData{
								Target:       target,
								Result:       result,
								Stats:        stats.Stats{},
								TargetKey:    targetKey,
								WebhookError: err,
							}
						}
					}
				}

				stats := monitor.GetStats()
				dataChannel <- TargetData{
					Target:    target,
					Result:    result,
					Stats:     stats,
					TargetKey: targetKey,
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
