package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
)

func encodeAndPrint(data interface{}, writer io.Writer) {
	var buf strings.Builder
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(data); err != nil {
		return
	}
	fmt.Fprint(writer, buf.String())
}

type MetricsData struct {
	Type           string    `json:"type"`
	Timestamp      time.Time `json:"timestamp"`
	URL            string    `json:"url"`
	Region         string    `json:"region,omitempty"`
	Uptime         float64   `json:"uptime"`
	AvgResponseMS  int64     `json:"avg_response_time_ms"`
	MinResponseMS  int64     `json:"min_response_time_ms"`
	MaxResponseMS  int64     `json:"max_response_time_ms"`
	P95ResponseMS  int64     `json:"p95_response_time_ms,omitempty"`
	ChecksCount    int       `json:"checks_count"`
	SuccessCount   int       `json:"success_count"`
	SuccessPercent float64   `json:"success_percent"`
}

type ErrorData struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	URL       string    `json:"url"`
	Region    string    `json:"region,omitempty"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Error     string    `json:"error,omitempty"`
}

type CheckData struct {
	Type            string              `json:"type"`
	Timestamp       time.Time           `json:"timestamp"`
	URL             string              `json:"url"`
	Region          string              `json:"region,omitempty"`
	ResolvedIP      string              `json:"resolved_ip,omitempty"`
	StatusCode      int                 `json:"status_code"`
	ResponseTimeMS  int64               `json:"response_time_ms"`
	Success         bool                `json:"success"`
	Method          string              `json:"method"`
	SequenceNum     int                 `json:"sequence_num"`
	RequestHeaders  map[string][]string `json:"request_headers,omitempty"`
	ResponseHeaders map[string][]string `json:"response_headers,omitempty"`
	RequestBody     string              `json:"request_body,omitempty"`
	ResponseBody    string              `json:"response_body,omitempty"`
	AssertionPassed bool                `json:"assertion_passed,omitempty"`
	AssertionText   string              `json:"assertion_text,omitempty"`
}

func LogMetrics(stats *stats.Stats, url string, region ...string) {
	if stats == nil {
		return
	}

	data := MetricsData{
		Type:           "metrics",
		Timestamp:      time.Now(),
		URL:            url,
		Uptime:         stats.UptimePercent,
		AvgResponseMS:  stats.AvgResponseTime.Milliseconds(),
		MinResponseMS:  stats.MinResponseTime.Milliseconds(),
		MaxResponseMS:  stats.MaxResponseTime.Milliseconds(),
		ChecksCount:    stats.ChecksCount,
		SuccessCount:   stats.SuccessCount,
		SuccessPercent: 0,
	}

	if len(region) > 0 && region[0] != "" {
		data.Region = region[0]
	}

	if stats.ChecksCount > 0 {
		data.SuccessPercent = float64(stats.SuccessCount) / float64(stats.ChecksCount) * 100
	}

	if stats.ChecksCount >= 2 && stats.P95 > 0 {
		data.P95ResponseMS = stats.P95.Milliseconds()
	}

	encodeAndPrint(data, os.Stdout)
}

func LogCheck(result net.WebsiteCheckResult, seq int, jsonFormat string, region ...string) {
	data := CheckData{
		Type:            "check",
		Timestamp:       result.LastCheckTime,
		URL:             result.URL,
		ResolvedIP:      result.ResolvedIP,
		StatusCode:      result.StatusCode,
		ResponseTimeMS:  result.ResponseTime.Milliseconds(),
		Success:         result.IsUp,
		Method:          result.Method,
		SequenceNum:     seq,
		AssertionPassed: result.AssertionPassed,
		AssertionText:   result.AssertText,
		RequestHeaders:  result.RequestHeaders,
		ResponseHeaders: result.ResponseHeaders,
		RequestBody:     result.RequestBody,
		ResponseBody:    result.ResponseBody,
	}

	if len(region) > 0 && region[0] != "" {
		data.Region = region[0]
	}

	encodeAndPrint(data, os.Stdout)
}

func LogError(url string, msg string, err error, region ...string) {
	data := ErrorData{
		Type:      "error",
		Timestamp: time.Now(),
		URL:       url,
		Level:     "error",
		Message:   msg,
	}

	if len(region) > 0 && region[0] != "" {
		data.Region = region[0]
	}

	if err != nil {
		data.Error = err.Error()
	}

	encodeAndPrint(data, os.Stderr)
}

func LogWarning(url string, msg string, region ...string) {
	data := ErrorData{
		Type:      "warning",
		Timestamp: time.Now(),
		URL:       url,
		Level:     "warning",
		Message:   msg,
	}

	if len(region) > 0 && region[0] != "" {
		data.Region = region[0]
	}

	encodeAndPrint(data, os.Stderr)
}
