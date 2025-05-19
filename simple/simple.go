package simple

import (
	"fmt"
	"strings"

	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
)

type OutputManager struct {
	URL string
}

func NewOutputManager(url string) *OutputManager {
	return &OutputManager{
		URL: url,
	}
}

func (o *OutputManager) PrintHeader() {
	fmt.Printf("UPDO %s:\n", o.URL)

	if strings.HasPrefix(o.URL, "https://") {
		sslDaysRemaining := net.GetSSLCertExpiry(o.URL)
		if sslDaysRemaining > 0 {
			fmt.Printf("SSL certificate expires in %d days\n", sslDaysRemaining)
		}
	}
}

func (o *OutputManager) PrintResult(result net.WebsiteCheckResult, monitor *stats.Monitor) {
	stats := monitor.GetStats()

	statusInfo := fmt.Sprintf("status=%d", result.StatusCode)
	if !result.IsUp {
		statusInfo = fmt.Sprintf("status=%d (DOWN)", result.StatusCode)
	}

	if result.AssertText != "" && !result.AssertionPassed {
		statusInfo += " (assertion failed)"
	}

	ipInfo := ""
	if result.ResolvedIP != "" {
		ipInfo = fmt.Sprintf(" from %s", result.ResolvedIP)
	}

	fmt.Printf("Response%s: seq=%d time=%dms %s uptime=%.1f%%\n",
		ipInfo,
		monitor.ChecksCount-1,
		result.ResponseTime.Milliseconds(),
		statusInfo,
		stats.UptimePercent)
}

func (o *OutputManager) PrintStatistics(stats *stats.Stats) {
	fmt.Printf("\n--- %s statistics ---\n", o.URL)
	fmt.Printf("%d checks, %d successful (%.1f%%)\n",
		stats.ChecksCount,
		stats.SuccessCount,
		stats.UptimePercent)

	if stats.ChecksCount > 0 {
		responseTimeStr := fmt.Sprintf("response time min/avg/max/stddev = %d/%d/%d/%.1f ms",
			stats.MinResponseTime.Milliseconds(),
			stats.AvgResponseTime.Milliseconds(),
			stats.MaxResponseTime.Milliseconds(),
			stats.StdDev)

		if stats.ChecksCount >= 2 && stats.P95 > 0 {
			responseTimeStr += fmt.Sprintf(", 95th percentile: %d ms", stats.P95.Milliseconds())
		}

		fmt.Println(responseTimeStr)
	}
}
