package inspect

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Owloops/updo/net"
	"github.com/charmbracelet/lipgloss"
)

var (
	methodStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1")).Bold(true) // Green
	urlStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA")).Bold(true) // Blue
	statusOkStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1")).Bold(true) // Green
	statusErrStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8")).Bold(true) // Red
	headerNameStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAB387"))            // Orange
	headerValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F5E0DC"))            // Light
	bodyStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4"))            // Light blue
	timingStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#F9E2AF"))            // Yellow
	labelStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#CBA6F7")).Bold(true) // Purple

	requestBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#A6E3A1")).
			Padding(1, 2).
			MarginBottom(1)

	responseBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#89B4FA")).
			Padding(1, 2).
			MarginBottom(1)

	timingBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#F9E2AF")).
			Padding(1, 2)
)

func formatInspectionResult(result *net.InspectionResult, config Config) string {
	var sections []string

	if config.PrintHeaders {
		sections = append(sections, formatRequestSection(result))
	}

	sections = append(sections, formatResponseSection(result, config))

	if config.Verbose {
		sections = append(sections, formatTimingSection(result))
	}

	return strings.Join(sections, "\n") + "\n"
}

func formatRequestSection(result *net.InspectionResult) string {
	var content strings.Builder

	requestLine := fmt.Sprintf("%s %s",
		methodStyle.Render(result.Method),
		urlStyle.Render(result.URL))
	content.WriteString(requestLine + "\n\n")

	if len(result.RequestHeaders) > 0 {
		content.WriteString(labelStyle.Render("Request Headers:") + "\n")
		headers := make([]string, 0, len(result.RequestHeaders))
		for name := range result.RequestHeaders {
			headers = append(headers, name)
		}
		sort.Strings(headers)

		for _, name := range headers {
			values := result.RequestHeaders[name]
			for _, value := range values {
				content.WriteString(fmt.Sprintf("%s: %s\n",
					headerNameStyle.Render(name),
					headerValueStyle.Render(value)))
			}
		}
	}

	if result.RequestBody != "" {
		content.WriteString("\n" + labelStyle.Render("Request Body:") + "\n")
		content.WriteString(bodyStyle.Render(result.RequestBody))
	}

	return requestBox.Render(content.String())
}

func formatResponseSection(result *net.InspectionResult, config Config) string {
	var content strings.Builder

	statusStyle := statusOkStyle
	if result.StatusCode >= 400 {
		statusStyle = statusErrStyle
	}

	statusLine := fmt.Sprintf("%s %s",
		statusStyle.Render(fmt.Sprintf("HTTP/%s", result.HTTPVersion)),
		statusStyle.Render(result.StatusText))
	content.WriteString(statusLine + "\n\n")

	if config.PrintHeaders && len(result.ResponseHeaders) > 0 {
		content.WriteString(labelStyle.Render("Response Headers:") + "\n")
		headers := make([]string, 0, len(result.ResponseHeaders))
		for name := range result.ResponseHeaders {
			headers = append(headers, name)
		}
		sort.Strings(headers)

		for _, name := range headers {
			values := result.ResponseHeaders[name]
			for _, value := range values {
				content.WriteString(fmt.Sprintf("%s: %s\n",
					headerNameStyle.Render(name),
					headerValueStyle.Render(value)))
			}
		}
	}

	if config.PrintBody && result.ResponseBody != "" {
		content.WriteString("\n" + labelStyle.Render("Response Body:") + "\n")
		formattedBody := formatResponseBody(result.ResponseBody, result.ResponseHeaders.Get("Content-Type"))
		content.WriteString(bodyStyle.Render(formattedBody))
	}

	return responseBox.Render(content.String())
}

func formatTimingSection(result *net.InspectionResult) string {
	var content strings.Builder

	content.WriteString(labelStyle.Render("Timing Breakdown:") + "\n\n")

	timings := []struct {
		label string
		value time.Duration
	}{
		{"DNS Lookup", result.TraceInfo.DNSLookup},
		{"TCP Connect", result.TraceInfo.TCPConnection},
		{"TLS Handshake", result.TraceInfo.Wait},
		{"Server Processing", result.TraceInfo.TimeToFirstByte},
		{"Content Download", result.TraceInfo.DownloadDuration},
		{"Total Time", result.ResponseTime},
	}

	maxLabelLen := 0
	for _, timing := range timings {
		if len(timing.label) > maxLabelLen {
			maxLabelLen = len(timing.label)
		}
	}

	for i, timing := range timings {
		label := fmt.Sprintf("%-*s", maxLabelLen, timing.label)
		value := formatDuration(timing.value)

		if i == len(timings)-1 {
			content.WriteString(fmt.Sprintf("%s: %s\n",
				labelStyle.Render(label),
				statusOkStyle.Render(value)))
		} else {
			content.WriteString(fmt.Sprintf("%s: %s\n",
				headerNameStyle.Render(label),
				timingStyle.Render(value)))
		}
	}

	return timingBox.Render(content.String())
}

func formatResponseBody(body, contentType string) string {
	if strings.Contains(contentType, "application/json") {
		var jsonObj interface{}
		if err := json.Unmarshal([]byte(body), &jsonObj); err == nil {
			if formatted, err := json.MarshalIndent(jsonObj, "", "  "); err == nil {
				return string(formatted)
			}
		}
	}

	if len(body) > 2000 {
		return body[:2000] + "\n\n" + headerNameStyle.Render("... (truncated)")
	}

	return body
}

func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%.0fns", float64(d.Nanoseconds()))
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.1fμs", float64(d.Nanoseconds())/1000.0)
	} else if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1000000.0)
	} else {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}
