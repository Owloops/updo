package simple

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
	"github.com/Owloops/updo/utils"
)

type OutputManager struct {
	targets      []config.Target
	isSingle     bool
	sslExpiry    map[string]int
	sslExpiryMu  sync.RWMutex
	sslCollected map[string]bool
}

func NewOutputManager(targets []config.Target) *OutputManager {
	return &OutputManager{
		targets:      targets,
		isSingle:     len(targets) == 1,
		sslExpiry:    make(map[string]int),
		sslCollected: make(map[string]bool),
	}
}

func (m *OutputManager) PrintHeader() {
	if m.isSingle {
		fmt.Printf("UPDO %s:\n", m.targets[0].URL)
	} else {
		fmt.Println("UPDO monitoring:")
		for _, target := range m.targets {
			fmt.Printf("%s: %s\n", target.Name, target.URL)
		}
	}

	m.startSSLCollection()
}

func (m *OutputManager) startSSLCollection() {
	for _, target := range m.targets {
		go func(url string) {
			if strings.HasPrefix(url, "https://") {
				sslDaysRemaining := net.GetSSLCertExpiry(url)
				m.sslExpiryMu.Lock()
				m.sslExpiry[url] = sslDaysRemaining
				m.sslCollected[url] = true
				m.sslExpiryMu.Unlock()
			}
		}(target.URL)
	}
}

func (m *OutputManager) getSSLExpiry(url string) int {
	m.sslExpiryMu.RLock()
	defer m.sslExpiryMu.RUnlock()
	if days, exists := m.sslExpiry[url]; exists {
		return days
	}
	return 0
}

func (m *OutputManager) PrintResult(result TargetResult) {
	statusInfo := fmt.Sprintf("status=%d", result.Result.StatusCode)
	if !result.Result.IsUp {
		statusInfo = fmt.Sprintf("status=%d (DOWN)", result.Result.StatusCode)
	}

	if result.Result.AssertText != "" && !result.Result.AssertionPassed {
		statusInfo += " (assertion failed)"
	}

	ipInfo := ""
	if result.Result.ResolvedIP != "" {
		ipInfo = fmt.Sprintf(" from %s", result.Result.ResolvedIP)
	}

	regionInfo := ""
	if result.Region != "" {
		regionInfo = fmt.Sprintf(" [%s]", result.Region)
	}

	if m.isSingle {
		fmt.Printf("Response%s%s: seq=%d time=%dms %s uptime=%.1f%%\n",
			ipInfo,
			regionInfo,
			result.Sequence,
			result.Result.ResponseTime.Milliseconds(),
			statusInfo,
			result.Stats.UptimePercent)
	} else {
		fmt.Printf("%s response%s%s: seq=%d time=%dms %s uptime=%.1f%%\n",
			result.Target.Name,
			ipInfo,
			regionInfo,
			result.Sequence,
			result.Result.ResponseTime.Milliseconds(),
			statusInfo,
			result.Stats.UptimePercent)
	}
}

func (m *OutputManager) PrintFinalStatistics(monitors map[string]*stats.Monitor, targets []config.Target, logMode bool) {
	if !logMode {
		m.PrintStatistics(monitors)
	} else {
		for _, target := range targets {
			stats := monitors[target.Name].GetStats()
			utils.LogMetrics(&stats, target.URL)
		}
	}
}

func (m *OutputManager) PrintFinalStatisticsWithKeys(monitors map[string]*stats.Monitor, keyRegistry *stats.TargetKeyRegistry, logMode bool) {
	if !logMode {
		m.PrintStatisticsWithKeys(monitors, keyRegistry)
	} else {
		targetMonitors := make(map[string]*stats.Monitor)
		for _, target := range m.targets {
			keys := keyRegistry.GetKeysForTarget(target.Name)
			if len(keys) > 0 {
				firstKey := keys[0]
				if monitor, exists := monitors[firstKey.String()]; exists {
					targetMonitors[target.Name] = monitor
				}
			}
		}

		for _, target := range m.targets {
			if monitor, exists := targetMonitors[target.Name]; exists {
				stats := monitor.GetStats()
				utils.LogMetrics(&stats, target.URL)
			}
		}
	}
}

func (m *OutputManager) PrintStatisticsWithKeys(monitors map[string]*stats.Monitor, keyRegistry *stats.TargetKeyRegistry) {
	if m.isSingle {
		target := m.targets[0]
		keys := keyRegistry.GetKeysForTarget(target.Name)
		if len(keys) == 0 {
			return
		}

		var aggregatedStats stats.Stats
		var hasStats bool

		for _, key := range keys {
			if monitor, exists := monitors[key.String()]; exists {
				keyStats := monitor.GetStats()
				if !hasStats {
					aggregatedStats = keyStats
					hasStats = true
				} else {
					aggregatedStats.ChecksCount += keyStats.ChecksCount
					aggregatedStats.SuccessCount += keyStats.SuccessCount
					if keyStats.MinResponseTime < aggregatedStats.MinResponseTime || aggregatedStats.MinResponseTime == 0 {
						aggregatedStats.MinResponseTime = keyStats.MinResponseTime
					}
					if keyStats.MaxResponseTime > aggregatedStats.MaxResponseTime {
						aggregatedStats.MaxResponseTime = keyStats.MaxResponseTime
					}
					if aggregatedStats.ChecksCount > 0 {
						aggregatedStats.UptimePercent = float64(aggregatedStats.SuccessCount) / float64(aggregatedStats.ChecksCount) * 100
					}
				}
			}
		}

		if hasStats {
			fmt.Printf("\n--- %s statistics ---\n", target.URL)

			successPercent := 0.0
			if aggregatedStats.ChecksCount > 0 {
				successPercent = float64(aggregatedStats.SuccessCount) / float64(aggregatedStats.ChecksCount) * 100
			}

			fmt.Printf("%d checks, %d successful (%.1f%%)\n",
				aggregatedStats.ChecksCount,
				aggregatedStats.SuccessCount,
				successPercent)

			fmt.Printf("uptime: %.1f%%\n", aggregatedStats.UptimePercent)

			if aggregatedStats.ChecksCount > 0 {
				var builder strings.Builder
				builder.Grow(100)
				builder.WriteString(fmt.Sprintf("response time min/avg/max/stddev = %d/%d/%d/%.1f ms",
					aggregatedStats.MinResponseTime.Milliseconds(),
					aggregatedStats.AvgResponseTime.Milliseconds(),
					aggregatedStats.MaxResponseTime.Milliseconds(),
					aggregatedStats.StdDev))

				if aggregatedStats.ChecksCount >= 2 && aggregatedStats.P95 > 0 {
					builder.WriteString(fmt.Sprintf(", 95th percentile: %d ms", aggregatedStats.P95.Milliseconds()))
				}

				fmt.Println(builder.String())
			}

			if sslDays := m.getSSLExpiry(target.URL); sslDays > 0 {
				fmt.Printf("SSL certificate expires in %d days\n", sslDays)
			}
		}
	} else {
		fmt.Println("\n--- statistics ---")
		allKeys := keyRegistry.GetAllKeys()
		for _, key := range allKeys {
			if key.TargetIndex >= 0 && key.TargetIndex < len(m.targets) {
				target := m.targets[key.TargetIndex]

				fmt.Printf("\n%s (%s):\n", target.Name, target.URL)

				if monitor, exists := monitors[key.String()]; exists {
					stats := monitor.GetStats()
					if !key.IsLocal {
						fmt.Printf("  Region [%s]:\n", key.Region)
						m.printTargetStatsIndented(stats, target.URL)
					} else {
						m.printTargetStats(stats, target.URL)
					}
				}
			}
		}
	}
}

func (m *OutputManager) printTargetStats(stats stats.Stats, url string) {
	successPercent := 0.0
	if stats.ChecksCount > 0 {
		successPercent = float64(stats.SuccessCount) / float64(stats.ChecksCount) * 100
	}

	fmt.Printf("  %d checks, %d successful (%.1f%%), uptime: %.1f%%\n",
		stats.ChecksCount, stats.SuccessCount, successPercent, stats.UptimePercent)

	if stats.ChecksCount > 0 {
		fmt.Printf("  response time min/avg/max = %d/%d/%d ms",
			stats.MinResponseTime.Milliseconds(),
			stats.AvgResponseTime.Milliseconds(),
			stats.MaxResponseTime.Milliseconds())

		if stats.ChecksCount >= 2 && stats.P95 > 0 {
			fmt.Printf(", 95p: %d ms", stats.P95.Milliseconds())
		}
		fmt.Println()
	}

	if sslDays := m.getSSLExpiry(url); sslDays > 0 {
		fmt.Printf("  SSL certificate expires in %d days\n", sslDays)
	}
}

func (m *OutputManager) printTargetStatsIndented(stats stats.Stats, url string) {
	successPercent := 0.0
	if stats.ChecksCount > 0 {
		successPercent = float64(stats.SuccessCount) / float64(stats.ChecksCount) * 100
	}

	fmt.Printf("    %d checks, %d successful (%.1f%%), uptime: %.1f%%\n",
		stats.ChecksCount, stats.SuccessCount, successPercent, stats.UptimePercent)

	if stats.ChecksCount > 0 {
		fmt.Printf("    response time min/avg/max = %d/%d/%d ms",
			stats.MinResponseTime.Milliseconds(),
			stats.AvgResponseTime.Milliseconds(),
			stats.MaxResponseTime.Milliseconds())

		if stats.ChecksCount >= 2 && stats.P95 > 0 {
			fmt.Printf(", 95p: %d ms", stats.P95.Milliseconds())
		}
		fmt.Println()
	}

	if sslDays := m.getSSLExpiry(url); sslDays > 0 {
		fmt.Printf("    SSL certificate expires in %d days\n", sslDays)
	}
}

func (m *OutputManager) PrintStatistics(monitors map[string]*stats.Monitor) {
	if m.isSingle {
		target := m.targets[0]
		monitor := monitors[target.Name]
		stats := monitor.GetStats()

		fmt.Printf("\n--- %s statistics ---\n", target.URL)

		successPercent := 0.0
		if stats.ChecksCount > 0 {
			successPercent = float64(stats.SuccessCount) / float64(stats.ChecksCount) * 100
		}

		fmt.Printf("%d checks, %d successful (%.1f%%)\n",
			stats.ChecksCount,
			stats.SuccessCount,
			successPercent)

		fmt.Printf("uptime: %.1f%%\n", stats.UptimePercent)

		if stats.ChecksCount > 0 {
			var builder strings.Builder
			builder.Grow(100)
			builder.WriteString(fmt.Sprintf("response time min/avg/max/stddev = %d/%d/%d/%.1f ms",
				stats.MinResponseTime.Milliseconds(),
				stats.AvgResponseTime.Milliseconds(),
				stats.MaxResponseTime.Milliseconds(),
				stats.StdDev))

			if stats.ChecksCount >= 2 && stats.P95 > 0 {
				builder.WriteString(fmt.Sprintf(", 95th percentile: %d ms", stats.P95.Milliseconds()))
			}

			fmt.Println(builder.String())
		}

		if sslDays := m.getSSLExpiry(target.URL); sslDays > 0 {
			fmt.Printf("SSL certificate expires in %d days\n", sslDays)
		}
	} else {
		fmt.Println("\n--- statistics ---")
		for _, target := range m.targets {
			monitor := monitors[target.Name]
			stats := monitor.GetStats()

			fmt.Printf("\n%s (%s):\n", target.Name, target.URL)

			successPercent := 0.0
			if stats.ChecksCount > 0 {
				successPercent = float64(stats.SuccessCount) / float64(stats.ChecksCount) * 100
			}

			fmt.Printf("  %d checks, %d successful (%.1f%%), uptime: %.1f%%\n",
				stats.ChecksCount, stats.SuccessCount, successPercent, stats.UptimePercent)

			if stats.ChecksCount > 0 {
				fmt.Printf("  response time min/avg/max = %d/%d/%d ms",
					stats.MinResponseTime.Milliseconds(),
					stats.AvgResponseTime.Milliseconds(),
					stats.MaxResponseTime.Milliseconds())

				if stats.ChecksCount >= 2 && stats.P95 > 0 {
					fmt.Printf(", 95p: %d ms", stats.P95.Milliseconds())
				}
				fmt.Println()
			}

			if sslDays := m.getSSLExpiry(target.URL); sslDays > 0 {
				fmt.Printf("  SSL certificate expires in %d days\n", sslDays)
			}
		}
	}
}
