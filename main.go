package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Owloops/updo/tui"

	"github.com/gen2brain/beeep"
	ui "github.com/gizak/termui/v3"
)

var (
	uptimeData        []float64
	responseTimeData  []float64
	totalUptime       time.Duration
	totalResponseTime time.Duration
	checksCount       int
	startTime         time.Time
	lastCheckTime     time.Time
	isUp              bool
	width             int
	height            int
)

func alert(message string) {
	err := beeep.Notify("Website Status Alert", message, "assets/information.png")
	if err != nil {
	}
}

func main() {
	urlFlag := flag.String("url", "https://example.com", "URL of the website to monitor")
	refreshFlag := flag.Int("refresh", 5, "Refresh interval in seconds")
	shouldFailFlag := flag.Bool("should-fail", false, "Invert status code success (200-299 are failures, 400+ are successes)")
	timeoutFlag := flag.Int("timeout", 10, "HTTP request timeout in seconds")
	followRedirectsFlag := flag.Bool("follow-redirects", true, "Follow redirects")
	skipSSLFlag := flag.Bool("skip-ssl", false, "Skip SSL certificate verification")
	assertTextFlag := flag.String("assert-text", "", "Text to assert in the response body")
	receiveAlertFlag := flag.Bool("receive-alert", true, "Enable alert notifications")
	helpFlag := flag.Bool("help", false, "Display this help message")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Println("This program monitors a website and displays various metrics.")
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  updo --url https://example.com --refresh=5 --should-fail=false --timeout=10")
		fmt.Println("  updo --url https://example.com --should-fail=true")
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	if !strings.HasPrefix(*urlFlag, "http://") && !strings.HasPrefix(*urlFlag, "https://") {
		log.Fatalf("Error: URL must start with http:// or https://")
	}

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	tui.InitializeWidgets()

	timeout := time.Duration(*timeoutFlag) * time.Second
	url := *urlFlag
	refreshInterval := time.Second * time.Duration(*refreshFlag)

	width, height = ui.TerminalDimensions()
	startTime = time.Now()
	lastCheckTime = startTime
	alertSent := false
	lastCheckTime, isUp = tui.PerformCheckAndUpdateWidgets(
		url, *shouldFailFlag, timeout, *followRedirectsFlag, *skipSSLFlag, *assertTextFlag, *refreshFlag, lastCheckTime, startTime, width, height,
	)
	if !isUp && !alertSent && *receiveAlertFlag {
		alert("The website is down!")
		alertSent = true
	} else if isUp && alertSent {
		if *receiveAlertFlag {
			alert("The website is back up!")
		}
		alertSent = false
	}

	ticker := time.NewTicker(refreshInterval)
	uiEvents := ui.PollEvents()

	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				tui.UpdateQuitWidgetText("Quitting...")
				return
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				width, height = payload.Width, payload.Height
			}
		case <-ticker.C:
			lastCheckTime, isUp = tui.PerformCheckAndUpdateWidgets(url, *shouldFailFlag, timeout, *followRedirectsFlag, *skipSSLFlag, *assertTextFlag, *refreshFlag, lastCheckTime, startTime, width, height)
			if !isUp && !alertSent && *receiveAlertFlag {
				alert("The website is down!")
				alertSent = true
			} else if isUp && alertSent {
				if *receiveAlertFlag {
					alert("The website is back up!")
				}
				alertSent = false
			}
		default:
			time.Sleep(time.Millisecond * 10)
		}
	}
}
