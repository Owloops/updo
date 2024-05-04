package main

import (
	"flag"
	"fmt"
	"log"
	"os"
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

func main() {
	config := parseFlags()
	if err := ui.Init(); err != nil {
		log.Fatalf("Failed to initialize termui: %v", err)
	}
	defer ui.Close()
	tuiManager := tui.NewManager()
	tuiManager.InitializeWidgets(config.URL, config.RefreshInterval)
	startMonitoring(config, tuiManager)
}

func parseFlags() AppConfig {
	var (
		urlFlag             string
		refreshFlag         int
		timeoutFlag         int
		shouldFailFlag      bool
		followRedirectsFlag bool
		skipSSLFlag         bool
		assertTextFlag      string
		receiveAlertFlag    bool
		helpFlag            bool
	)

	flag.StringVar(&urlFlag, "url", "", "URL or IP address to monitor")
	flag.StringVar(&urlFlag, "u", "", "Shorthand for -url")
	flag.IntVar(&refreshFlag, "refresh", 5, "Refresh interval in seconds")
	flag.IntVar(&refreshFlag, "r", 5, "Shorthand for -refresh")
	flag.IntVar(&timeoutFlag, "timeout", 10, "HTTP request timeout in seconds")
	flag.IntVar(&timeoutFlag, "t", 10, "Shorthand for -timeout")
	flag.BoolVar(&shouldFailFlag, "should-fail", false, "Invert success code range")
	flag.BoolVar(&shouldFailFlag, "f", false, "Shorthand for -should-fail")
	flag.BoolVar(&followRedirectsFlag, "follow-redirects", true, "Follow redirects")
	flag.BoolVar(&followRedirectsFlag, "l", true, "Shorthand for -follow-redirects")
	flag.BoolVar(&skipSSLFlag, "skip-ssl", false, "Skip SSL certificate verification")
	flag.BoolVar(&skipSSLFlag, "s", false, "Shorthand for -skip-ssl")
	flag.StringVar(&assertTextFlag, "assert-text", "", "Text to assert in response body")
	flag.StringVar(&assertTextFlag, "a", "", "Shorthand for -assert-text")
	flag.BoolVar(&receiveAlertFlag, "receive-alert", true, "Enable alert notifications")
	flag.BoolVar(&receiveAlertFlag, "n", true, "Shorthand for -receive-alert")
	flag.BoolVar(&helpFlag, "help", false, "Display this help message")
	flag.BoolVar(&helpFlag, "h", false, "Shorthand for -help")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Println("  updo [options] <URL/IP>")
		fmt.Println("Options:")
		fmt.Println("  -u, --url <URL/IP>             URL of the website or IP address to monitor")
		fmt.Println("  -r, --refresh <interval>       Refresh interval in seconds (default 5)")
		fmt.Println("  -t, --timeout <timeout>        HTTP request timeout in seconds (default 10)")
		fmt.Println("  -f, --should-fail              Invert status code success (200-299 are failures, 400+ are successes)")
		fmt.Println("  -l, --follow-redirects         Follow redirects (default true)")
		fmt.Println("  -s, --skip-ssl                 Skip SSL certificate verification")
		fmt.Println("  -a, --assert-text <text>       Text to assert in the response body")
		fmt.Println("  -n, --receive-alert            Enable alert notifications (default true)")
		fmt.Println("  -h, --help            		  Show this help page")
		fmt.Println("\nExamples:")
		fmt.Println("  updo --url https://example.com --refresh 5 --should-fail false --timeout 10")
		fmt.Println("  updo -u https://example.com -r 5 -f false -t 10")
	}

	flag.Parse()

	if helpFlag || urlFlag == "" && len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(0)
	}

	urlArg := urlFlag
	if urlArg == "" && len(flag.Args()) > 0 {
		urlArg = flag.Args()[0]
	}

	urlArg = net.AutoDetectProtocol(urlArg)

	return AppConfig{
		URL:             urlArg,
		RefreshInterval: time.Second * time.Duration(refreshFlag),
		Timeout:         time.Duration(timeoutFlag) * time.Second,
		ShouldFail:      shouldFailFlag,
		FollowRedirects: followRedirectsFlag,
		SkipSSL:         skipSSLFlag,
		AssertText:      assertTextFlag,
		ReceiveAlert:    receiveAlertFlag,
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

func alert(message string) {
	beeep.Notify("Website Status Alert", message, "assets/information.png")
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
