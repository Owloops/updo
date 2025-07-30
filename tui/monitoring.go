package tui

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Owloops/updo/aws"
	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/notifications"
	"github.com/Owloops/updo/stats"
	ui "github.com/gizak/termui/v3"
)

const (
	requestFailedMsg   = "Request failed"
	assertionFailedMsg = "Assertion failed"
)

type TargetData struct {
	Target config.Target
	Result net.WebsiteCheckResult
	Stats  stats.Stats
	Region string
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

	monitors := make(map[string]*stats.Monitor)
	sequences := make(map[string]*int)
	alertStates := make(map[string]*bool)
	webhookAlertStates := make(map[string]*bool)

	for _, target := range targets {
		monitor, err := stats.NewMonitor()
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize stats monitor for %s: %v", target.Name, err))
		}
		monitors[target.Name] = monitor
		seq := 0
		alert := false
		webhookAlert := false
		sequences[target.Name] = &seq
		alertStates[target.Name] = &alert
		webhookAlertStates[target.Name] = &webhookAlert
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataChannel := make(chan TargetData, len(targets)*2)
	var wg sync.WaitGroup

	for _, target := range targets {
		wg.Add(1)
		go func(t config.Target) {
			defer wg.Done()
			monitorTargetTUI(ctx, t, monitors[t.Name], sequences[t.Name], alertStates[t.Name], webhookAlertStates[t.Name], dataChannel, options, monitors, sequences, alertStates, webhookAlertStates)
		}(target)
	}

	go func() {
		wg.Wait()
		close(dataChannel)
	}()

	manager := NewRefactoredManager(targets, options.Regions)
	width, height := ui.TerminalDimensions()
	manager.InitializeLayout(width, height)

	uiRefreshTicker := time.NewTicker(1 * time.Second)
	defer uiRefreshTicker.Stop()

	uiEvents := ui.PollEvents()
	currentTargetIndex := 0

	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				cancel()
				return
			case "<Resize>":
				if payload, ok := e.Payload.(ui.Resize); ok {
					width, height = payload.Width, payload.Height
					manager.Resize(width, height)
					ui.Render(manager.GetGrid())
				}
			case "<Down>":
				totalItems := len(targets)
				if len(options.Regions) > 0 && len(targets) == 1 {
					totalItems = len(options.Regions)
				}
				if totalItems > 1 {
					manager.NavigateTargets(1, &currentTargetIndex, monitors)
					ui.Render(manager.GetGrid())
				}
			case "<Up>":
				totalItems := len(targets)
				if len(options.Regions) > 0 && len(targets) == 1 {
					totalItems = len(options.Regions)
				}
				if totalItems > 1 {
					manager.NavigateTargets(-1, &currentTargetIndex, monitors)
					ui.Render(manager.GetGrid())
				}
			case "<Enter>":
				if len(targets) > 1 || len(options.Regions) > 0 {
					manager.ToggleExpansion()
				}
			case "/":
				if len(targets) > 1 || len(options.Regions) > 0 {
					manager.ToggleSearch()
					if manager.GetListWidgetForSearch() != nil && manager.GetListWidgetForSearch().IsSearchMode() && manager.GetListWidgetForSearch().OnSearchChange != nil {
						indices := manager.GetListWidgetForSearch().GetFilteredIndices()
						manager.GetListWidgetForSearch().OnSearchChange(manager.GetListWidgetForSearch().GetQuery(), indices)
					}
					ui.Render(manager.GetGrid())
				}
			case "<Escape>":
				if manager.GetListWidgetForSearch() != nil && manager.GetListWidgetForSearch().IsSearchMode() {
					manager.ToggleSearch()
					if manager.GetListWidgetForSearch().OnSearchChange != nil {
						manager.GetListWidgetForSearch().OnSearchChange(manager.GetListWidgetForSearch().GetQuery(), manager.GetListWidgetForSearch().GetFilteredIndices())
					}
					ui.Render(manager.GetGrid())
				}
			case "<Backspace>", "<C-8>", "<Space>":
				if manager.GetListWidgetForSearch() != nil && manager.GetListWidgetForSearch().IsSearchMode() {
					manager.GetListWidgetForSearch().UpdateSearch(e.ID)
					ui.Render(manager.GetGrid())
				}
			default:
				if manager.GetListWidgetForSearch() != nil && manager.GetListWidgetForSearch().IsSearchMode() && len(e.ID) == 1 {
					manager.GetListWidgetForSearch().UpdateSearch(e.ID)
					ui.Render(manager.GetGrid())
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

func monitorTargetTUI(ctx context.Context, target config.Target, monitor *stats.Monitor, sequence *int, alertSent *bool, webhookAlertSent *bool, dataChannel chan<- TargetData, options Options, monitors map[string]*stats.Monitor, sequences map[string]*int, alertStates map[string]*bool, webhookAlertStates map[string]*bool) {
	ticker := time.NewTicker(target.GetRefreshInterval())
	defer ticker.Stop()

	makeRequest := func() {
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

		if len(options.Regions) > 0 {
			lambdaResults := aws.InvokeMultiRegion(target.URL, netConfig, options.Regions, options.Profile)
			for _, lambdaResult := range lambdaResults {
				if lambdaResult.Error != nil {
					log.Printf("Lambda invocation failed for region %s: %v", lambdaResult.Region, lambdaResult.Error)
					continue
				}

				regionKey := fmt.Sprintf("%s@%s", target.Name, lambdaResult.Region)
				regionMonitor, exists := monitors[regionKey]
				if !exists {
					monitor, err := stats.NewMonitor()
					if err != nil {
						log.Printf("Failed to create monitor for %s: %v", regionKey, err)
						continue
					}
					monitors[regionKey] = monitor
					seq := 0
					alert := false
					webhookAlert := false
					sequences[regionKey] = &seq
					alertStates[regionKey] = &alert
					webhookAlertStates[regionKey] = &webhookAlert
					regionMonitor = monitor
				}
				regionSequence := sequences[regionKey]
				regionAlertSent := alertStates[regionKey]
				regionWebhookAlertSent := webhookAlertStates[regionKey]

				regionMonitor.AddResult(lambdaResult.Result)
				*regionSequence++

				if target.ReceiveAlert {
					notifications.HandleAlerts(lambdaResult.Result.IsUp, regionAlertSent, target.Name, lambdaResult.Result.URL)
				}

				if target.WebhookURL != "" {
					errorMsg := ""
					if !lambdaResult.Result.IsUp {
						switch {
						case lambdaResult.Result.StatusCode > 0:
							errorMsg = fmt.Sprintf("Non-success status code: %d", lambdaResult.Result.StatusCode)
						case lambdaResult.Result.AssertText != "" && !lambdaResult.Result.AssertionPassed:
							errorMsg = assertionFailedMsg
						default:
							errorMsg = requestFailedMsg
						}
					}
					notifications.HandleWebhookAlert(target.WebhookURL, target.WebhookHeaders, lambdaResult.Result.IsUp, regionWebhookAlertSent, target.Name, lambdaResult.Result.URL, lambdaResult.Result.ResponseTime, lambdaResult.Result.StatusCode, errorMsg)
				}

				dataChannel <- TargetData{
					Target: target,
					Result: lambdaResult.Result,
					Stats:  regionMonitor.GetStats(),
					Region: lambdaResult.Region,
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
				errorMsg := ""
				if !result.IsUp {
					switch {
					case result.StatusCode > 0:
						errorMsg = fmt.Sprintf("Non-success status code: %d", result.StatusCode)
					case result.AssertText != "" && !result.AssertionPassed:
						errorMsg = assertionFailedMsg
					default:
						errorMsg = requestFailedMsg
					}
				}
				notifications.HandleWebhookAlert(target.WebhookURL, target.WebhookHeaders, result.IsUp, webhookAlertSent, target.Name, target.URL, result.ResponseTime, result.StatusCode, errorMsg)
			}

			dataChannel <- TargetData{
				Target: target,
				Result: result,
				Stats:  monitor.GetStats(),
				Region: "",
			}
		}
	}

	makeRequest()
	if options.Count > 0 && monitor.ChecksCount >= options.Count {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			makeRequest()
			if options.Count > 0 && monitor.ChecksCount >= options.Count {
				return
			}
		}
	}
}
