package tui

import (
	"time"

	"github.com/Owloops/updo/net"
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
}

func StartMonitoring(config Config) {
	if err := ui.Init(); err != nil {
		panic(err)
	}
	defer ui.Close()

	manager, err := NewManager()
	if err != nil {
		panic(err)
	}

	manager.InitializeWidgets(config.URL, config.RefreshInterval)

	width, height := ui.TerminalDimensions()
	dataChannel := make(chan net.WebsiteCheckResult)

	checksCount := 0
	go func() {
		for {
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
			dataChannel <- result
			checksCount++

			if config.Count > 0 && checksCount >= config.Count {
				close(dataChannel)
				return
			}

			time.Sleep(config.RefreshInterval)
		}
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
			manager.UpdateWidgets(data, width, height)
			if config.ReceiveAlert {
				utils.HandleAlerts(data.IsUp, &alertSent)
			}

		case <-uiRefreshTicker.C:
			manager.UpdateDurationWidgets(width, height)
		}
	}
}
