package tui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Owloops/updo/aws"
	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/notifications"
	"github.com/Owloops/updo/stats"
	ui "github.com/gizak/termui/v3"
)

type TargetData struct {
	Target    config.Target
	Result    net.WebsiteCheckResult
	Stats     stats.Stats
	TargetKey TargetKey
}

type Options struct {
	Count   int
	Log     string
	Regions []string
	Profile string
}

func StartMonitoring(targets []config.Target, options Options) {
	if len(targets) == 0 {
		panic("No targets provided")
	}

	if err := ui.Init(); err != nil {
		panic(err)
	}
	defer ui.Close()

	keyRegistry := NewTargetKeyRegistry(targets, options.Regions)
	allKeys := keyRegistry.GetAllKeys()

	monitors := make(map[string]*stats.Monitor)
	sequences := make(map[string]*int)
	alertStates := make(map[string]*bool)
	webhookAlertStates := make(map[string]*bool)

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

	dataChannel := make(chan TargetData, len(targets)*2)
	var wg sync.WaitGroup

	for _, target := range targets {
		wg.Add(1)
		go func(t config.Target) {
			defer wg.Done()
			monitorTargetTUI(ctx, t, monitors, sequences, alertStates, webhookAlertStates, dataChannel, options)
		}(target)
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
				if manager.listWidget != nil && manager.listWidget.IsHeaderAtIndex(manager.listWidget.SelectedRow) {
					groupID := manager.listWidget.GetGroupAtIndex(manager.listWidget.SelectedRow)
					if groupID != "" {
						manager.preserveHeaderSelection = groupID
						manager.listWidget.ToggleGroupCollapse(groupID)
						manager.updateTargetList()
						manager.preserveHeaderSelection = ""
						ui.Render(manager.grid)
					}
				} else if manager.detailsManager.LogsWidget != nil {
					func() {
						defer func() {
							if r := recover(); r != nil {
								_ = r
							}
						}()
						manager.detailsManager.LogsWidget.ToggleExpand()
					}()
					ui.Render(manager.grid)
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
			case "<Backspace>", "<C-8>", "<Space>":
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

		case <-uiRefreshTicker.C:
			manager.RefreshStats(monitors)
		}
	}
}

func monitorTargetTUI(ctx context.Context, target config.Target, monitors map[string]*stats.Monitor, sequences map[string]*int, alertStates map[string]*bool, webhookAlertStates map[string]*bool, dataChannel chan<- TargetData, options Options) {
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
					errorResult := net.WebsiteCheckResult{
						URL:           target.URL,
						IsUp:          false,
						StatusCode:    0,
						LastCheckTime: time.Now(),
					}

					targetKey := NewRegionTargetKey(target.Name, lambdaResult.Region)
					dataChannel <- TargetData{
						Target:    target,
						Result:    errorResult,
						Stats:     stats.Stats{},
						TargetKey: targetKey,
					}
					continue
				}

				targetKey := NewRegionTargetKey(target.Name, lambdaResult.Region)
				targetKeyStr := targetKey.String()

				if monitor, exists := monitors[targetKeyStr]; exists {
					monitor.AddResult(lambdaResult.Result)
					if sequence, exists := sequences[targetKeyStr]; exists {
						*sequence++
					}

					if target.ReceiveAlert {
						if alertSent, exists := alertStates[targetKeyStr]; exists {
							notifications.HandleAlerts(lambdaResult.Result.IsUp, alertSent, target.Name, lambdaResult.Result.URL)
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
							notifications.HandleWebhookAlert(target.WebhookURL, target.WebhookHeaders, lambdaResult.Result.IsUp, webhookAlertSent, target.Name, lambdaResult.Result.URL, lambdaResult.Result.ResponseTime, lambdaResult.Result.StatusCode, errorMsg)
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
			targetKey := NewLocalTargetKey(target.Name)
			targetKeyStr := targetKey.String()

			if monitor, exists := monitors[targetKeyStr]; exists {
				monitor.AddResult(result)
				if sequence, exists := sequences[targetKeyStr]; exists {
					*sequence++
				}

				if target.ReceiveAlert {
					if alertSent, exists := alertStates[targetKeyStr]; exists {
						notifications.HandleAlerts(result.IsUp, alertSent, target.Name, target.URL)
					}
				}

				if target.WebhookURL != "" {
					errorMsg := ""
					if !result.IsUp {
						errorMsg = fmt.Sprintf("Status code: %d", result.StatusCode)
					}
					if webhookAlertSent, exists := webhookAlertStates[targetKeyStr]; exists {
						notifications.HandleWebhookAlert(
							target.WebhookURL,
							target.WebhookHeaders,
							result.IsUp,
							webhookAlertSent,
							target.Name,
							target.URL,
							result.ResponseTime,
							result.StatusCode,
							errorMsg,
						)
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
