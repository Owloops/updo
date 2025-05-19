package net

import (
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

type WebsiteCheckResult struct {
	URL             string
	IsUp            bool
	ResponseTime    time.Duration
	TraceInfo       *HttpTraceInfo
	AssertionPassed bool
	LastCheckTime   time.Time
	AssertText      string
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
	RefreshInterval time.Duration
}

func CheckWebsite(url string, config NetworkConfig) WebsiteCheckResult {
	result := WebsiteCheckResult{
		URL:           url,
		LastCheckTime: time.Now(),
	}

	var start, connect, dnsStart, dnsDone, gotFirstByte time.Time
	trace := &httptrace.ClientTrace{
		DNSStart:             func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:              func(_ httptrace.DNSDoneInfo) { dnsDone = time.Now() },
		GotConn:              func(_ httptrace.GotConnInfo) { connect = time.Now() },
		GotFirstResponseByte: func() { gotFirstByte = time.Now() },
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipSSL || isIPAddress(url)},
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

	req, _ := http.NewRequest("GET", url, nil)
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	start = time.Now()
	resp, err := client.Do(req)
	result.ResponseTime = time.Since(start)

	if err != nil {
		return result
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result
	}
	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	if config.ShouldFail {
		success = !success
	}

	bodyString := string(body)
	result.AssertionPassed = true
	result.AssertText = config.AssertText
	if config.AssertText != "" {
		result.AssertionPassed = strings.Contains(bodyString, config.AssertText)
		if !result.AssertionPassed {
			success = false
		}
	}

	result.IsUp = success

	result.TraceInfo = &HttpTraceInfo{
		Wait:             dnsStart.Sub(start),
		DNSLookup:        dnsDone.Sub(dnsStart),
		TCPConnection:    connect.Sub(dnsDone),
		TimeToFirstByte:  gotFirstByte.Sub(connect),
		DownloadDuration: time.Since(gotFirstByte),
	}

	return result
}

func GetSSLCertExpiry(siteUrl string) int {
	u, err := url.Parse(siteUrl)
	if err != nil {
		return -1
	}

	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":443"
	}

	conn, err := tls.Dial("tcp", host, &tls.Config{})
	if err != nil {
		return -1
	}
	defer conn.Close()

	if len(conn.ConnectionState().PeerCertificates) > 0 {
		cert := conn.ConnectionState().PeerCertificates[0]
		daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)
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
	const defaultTimeout = 5 * time.Second
	client := http.Client{
		Timeout: defaultTimeout,
	}
	resp, err := client.Head(urlString)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
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
