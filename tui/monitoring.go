package tui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
	"github.com/Owloops/updo/utils"
	ui "github.com/gizak/termui/v3"
)

type TargetData struct {
	Target config.Target
	Result net.WebsiteCheckResult
	Stats  stats.Stats
}

type Options struct {
	Count int
	Log   string
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

	for _, target := range targets {
		monitor, err := stats.NewMonitor()
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize stats monitor for %s: %v", target.Name, err))
		}
		monitors[target.Name] = monitor
		seq := 0
		alert := false
		sequences[target.Name] = &seq
		alertStates[target.Name] = &alert
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataChannel := make(chan TargetData, len(targets)*2)
	var wg sync.WaitGroup

	for _, target := range targets {
		wg.Add(1)
		go func(t config.Target) {
			defer wg.Done()
			monitorTargetTUI(ctx, t, monitors[t.Name], sequences[t.Name], alertStates[t.Name], dataChannel, options)
		}(target)
	}

	go func() {
		wg.Wait()
		close(dataChannel)
	}()

	manager := NewManager(targets)
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
				}
			case "<Down>", "j":
				if len(targets) > 1 {
					currentTargetIndex = (currentTargetIndex + 1) % len(targets)
					manager.SetActiveTarget(currentTargetIndex, monitors)
				}
			case "<Up>", "k":
				if len(targets) > 1 {
					currentTargetIndex = (currentTargetIndex - 1 + len(targets)) % len(targets)
					manager.SetActiveTarget(currentTargetIndex, monitors)
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

func monitorTargetTUI(ctx context.Context, target config.Target, monitor *stats.Monitor, sequence *int, alertSent *bool, dataChannel chan<- TargetData, options Options) {
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

		result := net.CheckWebsite(target.URL, netConfig)
		monitor.AddResult(result)
		*sequence++

		if target.ReceiveAlert {
			utils.HandleAlerts(result.IsUp, alertSent)
		}

		stats := monitor.GetStats()
		dataChannel <- TargetData{
			Target: target,
			Result: result,
			Stats:  stats,
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
