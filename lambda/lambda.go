package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"maps"

	"github.com/Owloops/updo/net"
	"github.com/aws/aws-lambda-go/lambda"
)

const (
	_defaultTimeout = 10 * time.Second
	_maxTimeout     = 25 * time.Second
	_bufferTime     = 1 * time.Second
	_unknownRegion  = "unknown"
)

var (
	_awsRegion string
)

func init() {
	_awsRegion = os.Getenv("AWS_REGION")
	if _awsRegion == "" {
		_awsRegion = os.Getenv("AWS_DEFAULT_REGION")
	}
	if _awsRegion == "" {
		_awsRegion = _unknownRegion
		log.Printf("Warning: AWS region not found in environment variables")
	}
	log.Printf("Lambda function initialized in region: %s", _awsRegion)
}

type CheckRequest struct {
	URL             string   `json:"url"`
	Method          string   `json:"method"`
	Headers         []string `json:"headers"`
	Body            string   `json:"body"`
	Timeout         int      `json:"timeout"`
	FollowRedirects bool     `json:"follow_redirects"`
	AcceptRedirects bool     `json:"accept_redirects"`
	SkipSSL         bool     `json:"skip_ssl"`
	AssertText      string   `json:"assert_text"`
	ShouldFail      bool     `json:"should_fail"`
}

type CheckResponse struct {
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
	AssertionPassed bool                 `json:"assertion_passed"`
}

type HttpTraceInfoSimple struct {
	WaitMs             float64 `json:"wait_ms"`
	DNSLookupMs        float64 `json:"dns_lookup_ms"`
	TCPConnectionMs    float64 `json:"tcp_connection_ms"`
	TimeToFirstByteMs  float64 `json:"time_to_first_byte_ms"`
	DownloadDurationMs float64 `json:"download_duration_ms"`
}

func handleRequest(ctx context.Context, req CheckRequest) (CheckResponse, error) {
	if req.URL == "" {
		return CheckResponse{Region: _awsRegion}, fmt.Errorf("URL is required")
	}

	resp := CheckResponse{
		Region: _awsRegion,
	}

	timeout := time.Duration(req.Timeout) * time.Second
	if req.Timeout <= 0 {
		timeout = _defaultTimeout
	}
	if timeout > _maxTimeout {
		timeout = _maxTimeout
		log.Printf("Warning: Timeout capped at 25s to prevent Lambda timeout")
	}

	if deadline, ok := ctx.Deadline(); ok {
		remainingTime := time.Until(deadline) - _bufferTime
		if remainingTime > 0 && remainingTime < timeout {
			timeout = remainingTime
			log.Printf("Adjusting timeout to %v to respect Lambda deadline", timeout)
		}
	}

	netConfig := net.NetworkConfig{
		Timeout:         timeout,
		ShouldFail:      req.ShouldFail,
		FollowRedirects: req.FollowRedirects,
		AcceptRedirects: req.AcceptRedirects,
		SkipSSL:         req.SkipSSL,
		AssertText:      req.AssertText,
		Headers:         req.Headers,
		Method:          req.Method,
		Body:            req.Body,
	}

	result := net.CheckWebsite(req.URL, netConfig)

	resp.StatusCode = result.StatusCode
	resp.ResponseTimeMs = float64(result.ResponseTime / time.Millisecond)
	resp.ResolvedIP = result.ResolvedIP
	resp.RequestBody = result.RequestBody
	resp.ResponseBody = result.ResponseBody

	if result.RequestHeaders != nil {
		resp.RequestHeaders = make(map[string][]string, len(result.RequestHeaders))
		maps.Copy(resp.RequestHeaders, result.RequestHeaders)
	}
	if result.ResponseHeaders != nil {
		resp.ResponseHeaders = make(map[string][]string, len(result.ResponseHeaders))
		maps.Copy(resp.ResponseHeaders, result.ResponseHeaders)
	}

	if isHTTPS(req.URL) {
		checkSSLExpiry(req.URL, &resp)
	}

	resp.Success = result.IsUp
	resp.AssertionPassed = result.AssertionPassed

	if result.TraceInfo != nil {
		resp.TraceInfo = &HttpTraceInfoSimple{
			WaitMs:             float64(result.TraceInfo.Wait / time.Millisecond),
			DNSLookupMs:        float64(result.TraceInfo.DNSLookup / time.Millisecond),
			TCPConnectionMs:    float64(result.TraceInfo.TCPConnection / time.Millisecond),
			TimeToFirstByteMs:  float64(result.TraceInfo.TimeToFirstByte / time.Millisecond),
			DownloadDurationMs: float64(result.TraceInfo.DownloadDuration / time.Millisecond),
		}
	}

	if !result.IsUp && result.StatusCode == 0 {
		resp.Error = "connection failed: unable to reach host"
	}

	return resp, nil
}

func isHTTPS(url string) bool {
	return strings.HasPrefix(strings.ToLower(url), "https://")
}

func checkSSLExpiry(url string, resp *CheckResponse) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Warning: SSL check failed for %s: %v", url, r)
		}
	}()
	sslDays := net.GetSSLCertExpiry(url)
	if sslDays >= 0 {
		resp.SSLExpiry = &sslDays
	}
}

func main() {
	lambda.Start(handleRequest)
}
