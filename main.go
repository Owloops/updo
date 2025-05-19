package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/simple"
	"github.com/Owloops/updo/tui"
	"golang.org/x/term"
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
	Simple          bool
	Count           int
}

func main() {
	config := parseFlags()

	useSimpleMode := config.Simple || !term.IsTerminal(int(os.Stdout.Fd()))

	if useSimpleMode {
		simpleConfig := simple.Config{
			URL:             config.URL,
			RefreshInterval: config.RefreshInterval,
			Timeout:         config.Timeout,
			ShouldFail:      config.ShouldFail,
			FollowRedirects: config.FollowRedirects,
			SkipSSL:         config.SkipSSL,
			AssertText:      config.AssertText,
			ReceiveAlert:    config.ReceiveAlert,
			Count:           config.Count,
		}
		simple.StartMonitoring(simpleConfig)
	} else {
		tuiConfig := tui.Config{
			URL:             config.URL,
			RefreshInterval: config.RefreshInterval,
			Timeout:         config.Timeout,
			ShouldFail:      config.ShouldFail,
			FollowRedirects: config.FollowRedirects,
			SkipSSL:         config.SkipSSL,
			AssertText:      config.AssertText,
			ReceiveAlert:    config.ReceiveAlert,
			Count:           config.Count,
		}
		tui.StartMonitoring(tuiConfig)
	}
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
		simpleFlag          bool
		countFlag           int
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
	flag.BoolVar(&simpleFlag, "simple", false, "Use simple output instead of TUI")
	flag.IntVar(&countFlag, "count", 0, "Number of checks to perform (0 = infinite)")
	flag.IntVar(&countFlag, "c", 0, "Shorthand for -count")
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
		fmt.Println("     --simple                    Use simple output instead of TUI")
		fmt.Println("                                 (TUI mode will auto-fallback to simple mode when terminal is not supported)")
		fmt.Println("  -c, --count <count>            Number of checks to perform (0 = infinite)")
		fmt.Println("  -h, --help                     Show this help message")
		fmt.Println("\nExamples:")
		fmt.Println("  updo https://example.com")
		fmt.Println("  updo -r 10 -t 5 https://example.com")
		fmt.Println("  updo --simple -c 10 https://example.com")
		fmt.Println("  updo -a \"Welcome\" https://example.com")
	}

	flag.Parse()

	if helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	if urlFlag == "" && flag.NArg() > 0 {
		urlFlag = flag.Arg(0)
	}

	if urlFlag == "" {
		fmt.Println("Error: URL is required")
		fmt.Println("Use -h or --help for usage information")
		os.Exit(1)
	}

	urlFlag = net.AutoDetectProtocol(urlFlag)

	return AppConfig{
		URL:             urlFlag,
		RefreshInterval: time.Duration(refreshFlag) * time.Second,
		Timeout:         time.Duration(timeoutFlag) * time.Second,
		ShouldFail:      shouldFailFlag,
		FollowRedirects: followRedirectsFlag,
		SkipSSL:         skipSSLFlag,
		AssertText:      assertTextFlag,
		ReceiveAlert:    receiveAlertFlag,
		Simple:          simpleFlag,
		Count:           countFlag,
	}
}
