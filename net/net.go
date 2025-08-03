package net

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	_hoursPerDay    = 24
	_defaultTimeout = 5 * time.Second
	_userAgent      = "updo/1.0"
	_httpsPort      = ":443"
)

func parseHeaders(headers []string) map[string]string {
	headerMap := make(map[string]string, len(headers))
	for _, header := range headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headerMap[key] = value
		}
	}
	return headerMap
}

type WebsiteCheckResult struct {
	URL             string
	ResolvedIP      string
	IsUp            bool
	StatusCode      int
	ResponseTime    time.Duration
	TraceInfo       *HttpTraceInfo
	AssertionPassed bool
	LastCheckTime   time.Time
	AssertText      string
	Method          string
	RequestHeaders  http.Header
	ResponseHeaders http.Header
	RequestBody     string
	ResponseBody    string
}
type HttpTraceInfo struct {
	Wait             time.Duration
	DNSLookup        time.Duration
	TCPConnection    time.Duration
	TimeToFirstByte  time.Duration
	DownloadDuration time.Duration
}

type NetworkConfig struct {
	Timeout         time.Duration
	ShouldFail      bool
	FollowRedirects bool
	SkipSSL         bool
	AssertText      string
	Headers         []string
	Method          string
	Body            string
}

type HTTPRequestOptions struct {
	Method  string
	Headers map[string]string
	Body    string
}

type HTTPResponse struct {
	URL             string
	ResolvedIP      string
	StatusCode      int
	StatusText      string
	HTTPVersion     string
	ResponseHeaders http.Header
	ResponseBody    string
	RequestHeaders  http.Header
	RequestBody     string
	Method          string
	ResponseTime    time.Duration
	TraceInfo       *HttpTraceInfo
	LastCheckTime   time.Time
	Error           error
}

func CheckWebsite(urlStr string, config NetworkConfig) WebsiteCheckResult {
	method := "GET"
	if config.Method != "" {
		method = config.Method
	}

	options := HTTPRequestOptions{
		Method:  method,
		Headers: make(map[string]string, len(config.Headers)),
		Body:    config.Body,
	}

	if len(config.Headers) > 0 {
		for key, value := range parseHeaders(config.Headers) {
			options.Headers[key] = value
		}
	}

	httpResp := makeHTTPRequest(urlStr, options, config)

	result := WebsiteCheckResult{
		URL:             urlStr,
		ResolvedIP:      httpResp.ResolvedIP,
		StatusCode:      httpResp.StatusCode,
		ResponseTime:    httpResp.ResponseTime,
		TraceInfo:       httpResp.TraceInfo,
		LastCheckTime:   httpResp.LastCheckTime,
		AssertText:      config.AssertText,
		Method:          options.Method,
		RequestHeaders:  httpResp.RequestHeaders,
		ResponseHeaders: httpResp.ResponseHeaders,
		RequestBody:     httpResp.RequestBody,
		ResponseBody:    httpResp.ResponseBody,
	}

	if httpResp.Error != nil {
		result.Method = options.Method
		return result
	}

	success := httpResp.StatusCode >= 200 && httpResp.StatusCode < 300
	if config.ShouldFail {
		success = !success
	}

	result.AssertionPassed = true
	if config.AssertText != "" {
		result.AssertionPassed = strings.Contains(httpResp.ResponseBody, config.AssertText)
		if !result.AssertionPassed {
			success = false
		}
	}

	result.IsUp = success
	return result
}

func GetSSLCertExpiry(siteURL string) int {
	u, err := url.Parse(siteURL)
	if err != nil {
		return -1
	}

	host := u.Host
	if !strings.Contains(host, ":") {
		host += _httpsPort
	}

	conn, err := tls.Dial("tcp", host, &tls.Config{
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return -1
	}
	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Printf("Warning: failed to close TLS connection: %v\n", err)
		}
	}()

	if len(conn.ConnectionState().PeerCertificates) > 0 {
		cert := conn.ConnectionState().PeerCertificates[0]
		daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / _hoursPerDay)
		return daysUntilExpiry
	}

	return -1
}

func isIPAddress(host string) bool {
	u, err := url.Parse(host)
	if err != nil {
		return false
	}
	hostname := u.Hostname()

	return net.ParseIP(hostname) != nil
}

func TryHTTPSConnection(urlString string) (*http.Response, error) {
	client := http.Client{
		Timeout: _defaultTimeout,
	}
	resp, err := client.Head(urlString)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()
	return resp, nil
}

func formatURL(inputURL string) (string, error) {
	inputURL = strings.TrimSpace(inputURL)
	if strings.Contains(inputURL, "://") {
		return inputURL, nil
	}
	return "https://" + inputURL, nil
}

func isUrl(str string) bool {
	u, err := url.ParseRequestURI(str)
	if err != nil {
		return false
	}

	hostname := u.Hostname()
	address := net.ParseIP(hostname)

	return address != nil || hostname != ""
}

func AutoDetectProtocol(inputURL string) string {
	formattedURL, err := formatURL(inputURL)
	if err != nil {
		log.Printf("Error normalizing URL: %v, fallback to input URL\n", err)
		return inputURL
	}
	if !isUrl(formattedURL) {
		fmt.Printf("Error: Invalid URL provided. Please ensure the URL is correct.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	resp, err := TryHTTPSConnection(formattedURL)
	if err == nil && resp.StatusCode < 400 {
		return formattedURL
	}

	if strings.HasPrefix(formattedURL, "https://") {
		fallbackURL := strings.Replace(formattedURL, "https://", "http://", 1)
		resp, err := TryHTTPSConnection(fallbackURL)
		if err == nil && resp.StatusCode < 400 {
			log.Println("Fallback to HTTP successful.")
			return fallbackURL
		}
	}

	return formattedURL
}

func makeHTTPRequest(urlStr string, options HTTPRequestOptions, config NetworkConfig) *HTTPResponse {
	result := &HTTPResponse{
		URL:            urlStr,
		Method:         options.Method,
		RequestBody:    options.Body,
		LastCheckTime:  time.Now(),
		RequestHeaders: make(http.Header),
	}

	if parsedURL, parseErr := url.Parse(urlStr); parseErr == nil {
		if ips, lookupErr := net.LookupIP(parsedURL.Hostname()); lookupErr == nil && len(ips) > 0 {
			result.ResolvedIP = ips[0].String()
		}
	}

	var start, connect, dnsStart, dnsDone, gotFirstByte time.Time
	trace := &httptrace.ClientTrace{
		DNSStart:             func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:              func(_ httptrace.DNSDoneInfo) { dnsDone = time.Now() },
		GotConn:              func(_ httptrace.GotConnInfo) { connect = time.Now() },
		GotFirstResponseByte: func() { gotFirstByte = time.Now() },
	}

	transport := &http.Transport{
		// #nosec G402 - InsecureSkipVerify is intentionally configurable for testing and IP addresses
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipSSL || isIPAddress(urlStr)},
	}

	client := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}
	if !config.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	var reqBody io.Reader
	if options.Body != "" {
		reqBody = bytes.NewBufferString(options.Body)
	}

	req, err := http.NewRequest(options.Method, urlStr, reqBody)
	if err != nil {
		result.Error = err
		return result
	}

	for name, value := range options.Headers {
		req.Header.Set(name, value)
		result.RequestHeaders.Set(name, value)
	}

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", _userAgent)
		result.RequestHeaders.Set("User-Agent", _userAgent)
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	start = time.Now()
	resp, err := client.Do(req)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Error = err
		return result
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = err
		return result
	}

	result.StatusCode = resp.StatusCode
	result.StatusText = resp.Status
	result.HTTPVersion = resp.Proto
	result.ResponseHeaders = resp.Header
	result.ResponseBody = string(bodyBytes)

	result.TraceInfo = &HttpTraceInfo{
		Wait:             dnsStart.Sub(start),
		DNSLookup:        dnsDone.Sub(dnsStart),
		TCPConnection:    connect.Sub(dnsDone),
		TimeToFirstByte:  gotFirstByte.Sub(connect),
		DownloadDuration: time.Since(gotFirstByte),
	}

	return result
}
