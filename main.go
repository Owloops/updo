package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/tui"
	"github.com/gen2brain/beeep"
	ui "github.com/gizak/termui/v3"
)

type AppConfig struct {
	URL             string
	RefreshInterval time.Duration
	Timeout         time.Duration
	ShouldFail      bool
	FollowRedirects bool
	SkipSSL         bool
	AssertText      string
	ReceiveAlert    bool
}

func alert(message string) {
	err := beeep.Notify("Website Status Alert", message, "assets/information.png")
	if err != nil {
		log.Println("Alert error:", err)
	}
}

func main() {
	config := parseFlags()

	if err := ui.Init(); err != nil {
		log.Fatalf("Failed to initialize termui: %v", err)
	}
	defer ui.Close()

	tuiManager := tui.NewManager()
	tuiManager.InitializeWidgets()

	startMonitoring(config, tuiManager)
}

func parseFlags() AppConfig {
	urlFlag := flag.String("url", "https://example.com", "URL of the website to monitor")
	refreshFlag := flag.Int("refresh", 5, "Refresh interval in seconds")
	shouldFailFlag := flag.Bool("should-fail", false, "Invert status code success (200-299 are failures, 400+ are successes)")
	timeoutFlag := flag.Int("timeout", 10, "HTTP request timeout in seconds")
	followRedirectsFlag := flag.Bool("follow-redirects", true, "Follow redirects")
	skipSSLFlag := flag.Bool("skip-ssl", false, "Skip SSL certificate verification")
	assertTextFlag := flag.String("assert-text", "", "Text to assert in the response body")
	receiveAlertFlag := flag.Bool("receive-alert", true, "Enable alert notifications")
	helpFlag := flag.Bool("help", false, "Display this help message")

	flag.Parse()

	if *helpFlag {
		fmt.Println("This program monitors a website and displays various metrics.")
		flag.Usage()
		fmt.Println("\nExamples:")
		fmt.Println("  updo --url https://example.com --refresh=5 --should-fail=false --timeout=10")
		fmt.Println("  updo --url https://example.com --should-fail=true")
		os.Exit(0)
	}

	if !strings.HasPrefix(*urlFlag, "http://") && !strings.HasPrefix(*urlFlag, "https://") {
		log.Fatalf("Error: URL must start with http:// or https://")
	}

	return AppConfig{
		URL:             *urlFlag,
		RefreshInterval: time.Second * time.Duration(*refreshFlag),
		Timeout:         time.Duration(*timeoutFlag) * time.Second,
		ShouldFail:      *shouldFailFlag,
		FollowRedirects: *followRedirectsFlag,
		SkipSSL:         *skipSSLFlag,
		AssertText:      *assertTextFlag,
		ReceiveAlert:    *receiveAlertFlag,
	}
}

func startMonitoring(config AppConfig, tuiManager *tui.Manager) {
	width, height := ui.TerminalDimensions()
	dataChannel := make(chan net.WebsiteCheckResult)

	go func() {
		for {
			netConfig := net.NetworkConfig{
				Timeout:         config.Timeout,
				ShouldFail:      config.ShouldFail,
				FollowRedirects: config.FollowRedirects,
				SkipSSL:         config.SkipSSL,
				AssertText:      config.AssertText,
				RefreshInterval: config.RefreshInterval,
			}
			result := net.CheckWebsite(config.URL, netConfig)
			dataChannel <- result
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

		case data := <-dataChannel:
			tuiManager.UpdateWidgets(data, width, height)
			if config.ReceiveAlert {
				handleAlerts(data.IsUp, &alertSent)
			}

		case <-uiRefreshTicker.C:
			tuiManager.UpdateDurationWidgets(width, height)
		}
	}
}

func handleAlerts(isUp bool, alertSent *bool) {
	if !isUp && !*alertSent {
		alert("The website is down!")
		*alertSent = true
	} else if isUp && *alertSent {
		alert("The website is back up!")
		*alertSent = false
	}
}
