package tui

import (
	"time"

	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
	"github.com/Owloops/updo/utils"
	ui "github.com/gizak/termui/v3"
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
	Headers         []string
	Method          string
	Body            string
	Log             string
}

func StartMonitoring(config Config) {
	if err := ui.Init(); err != nil {
		panic(err)
	}
	defer ui.Close()

	monitor, err := stats.NewMonitor()
	if err != nil {
		panic(err)
	}

	manager := NewManager()
	manager.InitializeWidgets(config.URL, config.RefreshInterval)

	width, height := ui.TerminalDimensions()
	dataChannel := make(chan net.WebsiteCheckResult)

	go func() {
		for monitor.ChecksCount < config.Count || config.Count == 0 {
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
			dataChannel <- result

			time.Sleep(config.RefreshInterval)
		}
		close(dataChannel)
	}()

	uiRefreshTicker := time.NewTicker(1 * time.Second)
	defer uiRefreshTicker.Stop()

	uiEvents := ui.PollEvents()
	alertSent := false

	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "<Resize>":
				width, height = e.Payload.(ui.Resize).Width, e.Payload.(ui.Resize).Height
			}

		case data, ok := <-dataChannel:
			if !ok {
				return
			}
			stats := monitor.GetStats()
			manager.UpdateWidgets(data, stats, width, height)
			if config.ReceiveAlert {
				utils.HandleAlerts(data.IsUp, &alertSent)
			}

		case <-uiRefreshTicker.C:
			stats := monitor.GetStats()
			manager.UpdateDurationWidgets(stats, width, height)
		}
	}
}
