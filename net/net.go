package net

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
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
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipSSL},
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

	conn, err := tls.Dial("tcp", u.Host+":443", &tls.Config{})
	if err != nil {
		return -1
	}
	defer conn.Close()

	cert := conn.ConnectionState().PeerCertificates[0]
	return int(cert.NotAfter.Sub(time.Now()).Hours() / 24)
}
