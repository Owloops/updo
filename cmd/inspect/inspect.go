package inspect

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Owloops/updo/cmd/root"
	"github.com/Owloops/updo/net"
	"github.com/spf13/cobra"
)

type Config struct {
	URL                 string
	Timeout             time.Duration
	ShouldFail          bool
	FollowRedirects     bool
	SkipSSL             bool
	AssertText          string
	Method              string
	Headers             map[string]string
	Body                string
	PrintHeaders        bool
	PrintBody           bool
	PrintRequestHeaders bool
	PrintRequestBody    bool
	Verbose             bool
	MaxOutput           int
}

var InspectCmd = &cobra.Command{
	Use:   "inspect [url]",
	Short: "Inspect a single HTTP request with detailed information",
	Long: `Inspect command performs a single HTTP request and displays detailed
information including headers, response body, timing breakdown, and status.
Similar to HTTPie but integrated with updo's monitoring capabilities.`,
	Example: `  updo inspect https://api.example.com
  updo inspect -X POST -d '{"key":"value"}' https://api.example.com
  updo inspect -H "Authorization: Bearer token" https://api.example.com
  updo inspect --print=HhBb https://example.com`,
	Run: func(cmd *cobra.Command, args []string) {
		config := buildInspectConfig(cmd, args)

		if config.URL == "" {
			fmt.Println("Error: URL is required")
			fmt.Println("Use updo inspect --help for usage information")
			os.Exit(1)
		}

		config.URL = net.AutoDetectProtocol(config.URL)

		result := performInspection(config)
		displayInspectionResult(result, config)
	},
}

func buildInspectConfig(cmd *cobra.Command, args []string) Config {
	rootConfig := root.AppConfig

	config := Config{
		Timeout:             rootConfig.Timeout,
		ShouldFail:          rootConfig.ShouldFail,
		FollowRedirects:     rootConfig.FollowRedirects,
		SkipSSL:             rootConfig.SkipSSL,
		AssertText:          rootConfig.AssertText,
		Headers:             make(map[string]string),
		PrintHeaders:        true,
		PrintBody:           true,
		PrintRequestHeaders: true,
		PrintRequestBody:    true,
		MaxOutput:           2000,
	}

	if len(args) > 0 && rootConfig.URL == "" {
		config.URL = args[0]
	} else {
		config.URL = rootConfig.URL
	}

	method, _ := cmd.Flags().GetString("request")
	config.Method = method

	headers, _ := cmd.Flags().GetStringSlice("header")
	for _, header := range headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) == 2 {
			config.Headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	body, _ := cmd.Flags().GetString("data")
	config.Body = body

	verbose, _ := cmd.Flags().GetBool("verbose")
	config.Verbose = verbose

	maxOutput, _ := cmd.Flags().GetInt("max-output")
	config.MaxOutput = maxOutput

	print, _ := cmd.Flags().GetString("print")
	if print != "" {
		config.PrintRequestHeaders = strings.Contains(print, "H")
		config.PrintRequestBody = strings.Contains(print, "B")
		config.PrintHeaders = strings.Contains(print, "h")
		config.PrintBody = strings.Contains(print, "b")
	}

	return config
}

func init() {
	InspectCmd.Flags().StringP("request", "X", "GET", "HTTP method to use")
	InspectCmd.Flags().StringSliceP("header", "H", []string{}, "HTTP headers (can be used multiple times)")
	InspectCmd.Flags().StringP("data", "d", "", "Request body data")
	InspectCmd.Flags().StringP("print", "p", "HhBb", "What to print: (H)request headers, (B)request body, (h)response headers, (b)response body")
	InspectCmd.Flags().BoolP("verbose", "v", false, "Show detailed timing information")
	InspectCmd.Flags().Int("max-output", 2000, "Maximum characters to display in response body (0 = no limit)")
}

func performInspection(config Config) *net.InspectionResult {
	netConfig := net.NetworkConfig{
		Timeout:         config.Timeout,
		ShouldFail:      config.ShouldFail,
		FollowRedirects: config.FollowRedirects,
		SkipSSL:         config.SkipSSL,
		AssertText:      config.AssertText,
	}

	return net.InspectRequest(config.URL, config.Method, config.Headers, config.Body, netConfig)
}

func displayInspectionResult(result *net.InspectionResult, config Config) {
	if result == nil {
		fmt.Println("Failed to perform inspection")
		return
	}

	output := formatInspectionResult(result, config)
	fmt.Print(output)
}
