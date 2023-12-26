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

type HttpTraceInfo struct {
	Wait             time.Duration
	DNSLookup        time.Duration
	TCPConnection    time.Duration
	TimeToFirstByte  time.Duration
	DownloadDuration time.Duration
}

func CheckWebsite(url string, shouldFail bool, timeout time.Duration, followRedirects bool, skipSSL bool, assertText string) (bool, time.Duration, *HttpTraceInfo, bool) {
	var start, connect, dnsStart, dnsDone, gotFirstByte time.Time

	trace := &httptrace.ClientTrace{
		DNSStart:             func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:              func(_ httptrace.DNSDoneInfo) { dnsDone = time.Now() },
		GotConn:              func(_ httptrace.GotConnInfo) { connect = time.Now() },
		GotFirstResponseByte: func() { gotFirstByte = time.Now() },
	}

	req, _ := http.NewRequest("GET", url, nil)
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSSL},
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	if !followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	start = time.Now()
	resp, err := client.Do(req)
	totalTime := time.Since(start)

	if err != nil {
		return false, totalTime, nil, false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, totalTime, nil, false
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	if shouldFail {
		success = !success
	}

	bodyString := string(body)
	assertionPassed := true
	if assertText != "" {
		assertionPassed = strings.Contains(bodyString, assertText)
		if !assertionPassed {
			success = false
		}
	}

	return success, totalTime, &HttpTraceInfo{
		Wait:             dnsStart.Sub(start),
		DNSLookup:        dnsDone.Sub(dnsStart),
		TCPConnection:    connect.Sub(dnsDone),
		TimeToFirstByte:  gotFirstByte.Sub(connect),
		DownloadDuration: time.Since(gotFirstByte),
	}, assertionPassed
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
