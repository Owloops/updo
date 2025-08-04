package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/Owloops/updo/net"
)

type LambdaRequest struct {
	URL             string   `json:"url"`
	Method          string   `json:"method"`
	Headers         []string `json:"headers"`
	Body            string   `json:"body"`
	Timeout         int      `json:"timeout"`
	FollowRedirects bool     `json:"follow_redirects"`
	SkipSSL         bool     `json:"skip_ssl"`
	AssertText      string   `json:"assert_text"`
	ShouldFail      bool     `json:"should_fail"`
}

type LambdaResponse struct {
	Success         bool                 `json:"success"`
	StatusCode      int                  `json:"status_code"`
	ResponseTimeMs  float64              `json:"response_time_ms"`
	Error           string               `json:"error,omitempty"`
	Region          string               `json:"region"`
	SSLExpiry       *int                 `json:"ssl_expiry_days,omitempty"`
	TraceInfo       *HttpTraceInfoSimple `json:"trace_info,omitempty"`
	ResolvedIP      string               `json:"resolved_ip,omitempty"`
	RequestHeaders  map[string][]string  `json:"request_headers,omitempty"`
	ResponseHeaders map[string][]string  `json:"response_headers,omitempty"`
	RequestBody     string               `json:"request_body,omitempty"`
	ResponseBody    string               `json:"response_body,omitempty"`
}

type HttpTraceInfoSimple struct {
	WaitMs             float64 `json:"wait_ms"`
	DNSLookupMs        float64 `json:"dns_lookup_ms"`
	TCPConnectionMs    float64 `json:"tcp_connection_ms"`
	TimeToFirstByteMs  float64 `json:"time_to_first_byte_ms"`
	DownloadDurationMs float64 `json:"download_duration_ms"`
}

type RegionResult struct {
	Region string
	Result net.WebsiteCheckResult
	Error  error
}

func InvokeMultiRegion(url string, config net.NetworkConfig, regions []string, profile string) []RegionResult {
	if len(regions) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	resultsChan := make(chan RegionResult, len(regions))

	for _, region := range regions {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			result := invokeLambdaInRegion(url, config, r, profile)
			resultsChan <- result
		}(region)
	}

	wg.Wait()
	close(resultsChan)

	results := make([]RegionResult, 0, len(regions))
	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}

func invokeLambdaInRegion(url string, config net.NetworkConfig, region string, profile string) RegionResult {
	ctx, cancel := context.WithTimeout(context.Background(), _awsOperationTimeout)
	defer cancel()

	cfg, err := loadAWSConfig(ctx, region, profile)
	if err != nil {
		return RegionResult{
			Region: region,
			Error:  fmt.Errorf("failed to load AWS config: %w", err),
		}
	}

	lambdaClient := lambda.NewFromConfig(cfg)

	request := LambdaRequest{
		URL:             url,
		Method:          config.Method,
		Headers:         config.Headers,
		Body:            config.Body,
		Timeout:         int(config.Timeout / time.Second),
		FollowRedirects: config.FollowRedirects,
		SkipSSL:         config.SkipSSL,
		AssertText:      config.AssertText,
		ShouldFail:      config.ShouldFail,
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return RegionResult{
			Region: region,
			Error:  fmt.Errorf("failed to marshal request: %w", err),
		}
	}

	functionName := fmt.Sprintf("%s-%s", _functionName, region)

	input := &lambda.InvokeInput{
		FunctionName: aws.String(functionName),
		Payload:      payload,
	}

	resp, err := lambdaClient.Invoke(ctx, input)
	if err != nil {
		return RegionResult{
			Region: region,
			Error:  fmt.Errorf("lambda invocation failed: %w", err),
		}
	}

	if resp.FunctionError != nil {
		return RegionResult{
			Region: region,
			Error:  fmt.Errorf("lambda function error: %s", *resp.FunctionError),
		}
	}

	var lambdaResp LambdaResponse
	if err := json.Unmarshal(resp.Payload, &lambdaResp); err != nil {
		return RegionResult{
			Region: region,
			Error:  fmt.Errorf("failed to unmarshal response: %w", err),
		}
	}

	if lambdaResp.Error != "" {
		return RegionResult{
			Region: region,
			Error:  fmt.Errorf("lambda returned error: %s", lambdaResp.Error),
		}
	}

	result := net.WebsiteCheckResult{
		URL:           url,
		IsUp:          lambdaResp.Success,
		StatusCode:    lambdaResp.StatusCode,
		ResponseTime:  time.Duration(lambdaResp.ResponseTimeMs) * time.Millisecond,
		LastCheckTime: time.Now(),
		Method:        request.Method,
		AssertText:    request.AssertText,
		ResolvedIP:    lambdaResp.ResolvedIP,
		RequestBody:   lambdaResp.RequestBody,
		ResponseBody:  lambdaResp.ResponseBody,
	}

	if lambdaResp.RequestHeaders != nil {
		result.RequestHeaders = http.Header(lambdaResp.RequestHeaders)
	}
	if lambdaResp.ResponseHeaders != nil {
		result.ResponseHeaders = http.Header(lambdaResp.ResponseHeaders)
	}

	if lambdaResp.TraceInfo != nil {
		result.TraceInfo = &net.HttpTraceInfo{
			Wait:             time.Duration(lambdaResp.TraceInfo.WaitMs) * time.Millisecond,
			DNSLookup:        time.Duration(lambdaResp.TraceInfo.DNSLookupMs) * time.Millisecond,
			TCPConnection:    time.Duration(lambdaResp.TraceInfo.TCPConnectionMs) * time.Millisecond,
			TimeToFirstByte:  time.Duration(lambdaResp.TraceInfo.TimeToFirstByteMs) * time.Millisecond,
			DownloadDuration: time.Duration(lambdaResp.TraceInfo.DownloadDurationMs) * time.Millisecond,
		}
	}

	if request.AssertText != "" {
		result.AssertionPassed = lambdaResp.Success
	} else {
		result.AssertionPassed = true
	}

	return RegionResult{
		Region: region,
		Result: result,
	}
}

func loadAWSConfig(ctx context.Context, region string, profile string) (aws.Config, error) {
	var opts []func(*config.LoadOptions) error

	opts = append(opts, config.WithRegion(region))

	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	return config.LoadDefaultConfig(ctx, opts...)
}
