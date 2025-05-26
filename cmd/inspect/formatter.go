package inspect

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Owloops/updo/net"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var (
	statusOkStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1")).Bold(true) // Green
	statusErrStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8")).Bold(true) // Red
	headerNameStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAB387"))            // Orange
	headerValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F5E0DC"))            // Light
	timingStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#F9E2AF"))            // Yellow
	labelStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#CBA6F7")).Bold(true) // Purple
)

func formatInspectionResult(result *net.InspectionResult, config Config) string {
	var output strings.Builder

	if config.PrintRequestHeaders {
		output.WriteString(formatHeaders(result.RequestHeaders))
	}
	if config.PrintRequestBody && result.RequestBody != "" {
		output.WriteString(formatBody(result.RequestBody, "application/json", config.MaxOutput) + "\n\n")
	}

	output.WriteString(formatStatusLine(result) + "\n")

	if config.PrintHeaders {
		output.WriteString(formatHeaders(result.ResponseHeaders))
	}
	output.WriteString("\n")

	if config.PrintBody && result.ResponseBody != "" {
		contentType := result.ResponseHeaders.Get("Content-Type")
		output.WriteString(formatBody(result.ResponseBody, contentType, config.MaxOutput) + "\n")
	}

	if config.Verbose {
		output.WriteString("\n" + formatTiming(result) + "\n")
	}

	return output.String()
}

func formatStatusLine(result *net.InspectionResult) string {
	style := statusOkStyle
	if result.StatusCode >= 400 {
		style = statusErrStyle
	}

	statusText := result.StatusText
	prefix := fmt.Sprintf("%d ", result.StatusCode)
	if strings.HasPrefix(statusText, prefix) {
		statusText = strings.TrimPrefix(statusText, prefix)
	}

	return style.Render(fmt.Sprintf("HTTP/%s %d %s", result.HTTPVersion, result.StatusCode, statusText))
}

func formatHeaders(headers map[string][]string) string {
	if len(headers) == 0 {
		return ""
	}

	var output strings.Builder
	names := getSortedHeaderNames(headers)

	for _, name := range names {
		for _, value := range headers[name] {
			output.WriteString(fmt.Sprintf("%s: %s\n",
				headerNameStyle.Render(name),
				headerValueStyle.Render(value)))
		}
	}
	output.WriteString("\n")
	return output.String()
}

func formatBody(body, contentType string, maxOutput int) string {
	if strings.Contains(contentType, "application/json") {
		formatted := formatJSON(body)
		if maxOutput > 0 && len(formatted) > maxOutput {
			return formatted[:maxOutput] + "\n\n" + headerNameStyle.Render("... (truncated)")
		}
		return formatted
	}

	if maxOutput > 0 && len(body) > maxOutput {
		return body[:maxOutput] + "\n\n" + headerNameStyle.Render("... (truncated)")
	}

	return body
}

func formatTiming(result *net.InspectionResult) string {
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

	maxWidth := getMaxLabelWidth(timings)
	var output strings.Builder

	for i, timing := range timings {
		label := fmt.Sprintf("%-*s", maxWidth, timing.label)
		duration := formatDuration(timing.value)

		if i == len(timings)-1 {
			output.WriteString(fmt.Sprintf("%s: %s",
				labelStyle.Render(label),
				statusOkStyle.Render(duration)))
		} else {
			output.WriteString(fmt.Sprintf("%s: %s\n",
				headerNameStyle.Render(label),
				timingStyle.Render(duration)))
		}
	}

	return output.String()
}

func formatJSON(body string) string {
	var jsonObj interface{}
	if err := json.Unmarshal([]byte(body), &jsonObj); err != nil {
		return body
	}

	formatted := reindentJSON(jsonObj)
	if formatted == "" {
		return body
	}

	highlighted := applyJSONSyntaxHighlighting(formatted)
	if highlighted == "" {
		return formatted
	}

	return highlighted
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%.0fns", float64(d.Nanoseconds()))
	case d < time.Millisecond:
		return fmt.Sprintf("%.1fμs", float64(d.Nanoseconds())/1000.0)
	case d < time.Second:
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1000000.0)
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

func getSortedHeaderNames(headers map[string][]string) []string {
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func getMaxLabelWidth(timings []struct {
	label string
	value time.Duration
}) int {
	maxWidth := 0
	for _, timing := range timings {
		if len(timing.label) > maxWidth {
			maxWidth = len(timing.label)
		}
	}
	return maxWidth
}

func reindentJSON(jsonObj interface{}) string {
	var buf strings.Builder
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(jsonObj); err != nil {
		return ""
	}

	return strings.TrimSpace(buf.String())
}

func applyJSONSyntaxHighlighting(jsonText string) string {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(120),
	)
	if err != nil {
		return ""
	}

	highlighted, err := renderer.Render("```json\n" + jsonText + "\n```")
	if err != nil {
		return ""
	}

	return strings.TrimSpace(highlighted)
}
