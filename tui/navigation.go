package tui

import (
	"fmt"
	"time"

	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
	"github.com/Owloops/updo/utils"
)

func (m *Manager) NavigateTargetKeys(direction int, monitors map[string]*stats.Monitor) {
	if m.listWidget == nil {
		return
	}

	allKeys := m.keyRegistry.GetAllKeys()
	if len(allKeys) == 0 {
		return
	}

	if m.listWidget.IsSearchMode() {
		m.navigateFilteredKeys(direction, allKeys)
	} else {
		m.navigateAllKeys(direction, allKeys)
	}

	m.updateActiveTarget(monitors)
}

func (m *Manager) navigateFilteredKeys(direction int, _ []TargetKey) {
	filteredIndices := m.listWidget.GetFilteredIndices()
	if len(filteredIndices) == 0 {
		return
	}

	currentFilteredIndex := -1
	for i, idx := range filteredIndices {
		if idx == m.currentKeyIndex {
			currentFilteredIndex = i
			break
		}
	}

	if currentFilteredIndex == -1 {
		if direction > 0 {
			currentFilteredIndex = 0
		} else {
			currentFilteredIndex = len(filteredIndices) - 1
		}
	} else {
		if direction > 0 {
			currentFilteredIndex = (currentFilteredIndex + 1) % len(filteredIndices)
		} else {
			currentFilteredIndex = (currentFilteredIndex - 1 + len(filteredIndices)) % len(filteredIndices)
		}
	}

	m.currentKeyIndex = filteredIndices[currentFilteredIndex]
	m.listWidget.SelectedRow = currentFilteredIndex
}

func (m *Manager) navigateAllKeys(direction int, allKeys []TargetKey) {
	if direction > 0 {
		m.currentKeyIndex = (m.currentKeyIndex + 1) % len(allKeys)
	} else {
		m.currentKeyIndex = (m.currentKeyIndex - 1 + len(allKeys)) % len(allKeys)
	}
	m.listWidget.SelectedRow = m.currentKeyIndex
}

func (m *Manager) updateActiveTarget(monitors map[string]*stats.Monitor) {
	allKeys := m.keyRegistry.GetAllKeys()
	if m.currentKeyIndex >= len(allKeys) {
		return
	}

	currentKey := allKeys[m.currentKeyIndex]

	var currentTarget *config.Target
	for _, target := range m.targets {
		if target.Name == currentKey.TargetName {
			currentTarget = &target
			break
		}
	}

	if currentTarget == nil {
		return
	}

	if !m.isSingle && m.listWidget != nil {
		m.listWidget.Title = "Targets → " + currentKey.DisplayName()
	}

	m.detailsManager.URLWidget.Text = currentTarget.URL
	m.detailsManager.RefreshWidget.Text = fmt.Sprintf("%v seconds", currentTarget.GetRefreshInterval().Seconds())

	m.restorePlotData(currentKey.String())
	m.updateTargetList()

	if monitor, exists := monitors[currentKey.String()]; exists {
		freshStats := monitor.GetStats()
		m.detailsManager.UptimeWidget.Text = fmt.Sprintf("%.2f%%", freshStats.UptimePercent)
		m.detailsManager.UpForWidget.Text = utils.FormatDurationMinute(freshStats.TotalDuration)
		m.detailsManager.AvgResponseTimeWidget.Text = utils.FormatDurationMillisecond(freshStats.AvgResponseTime)
		m.detailsManager.MinResponseTimeWidget.Text = utils.FormatDurationMillisecond(freshStats.MinResponseTime)
		m.detailsManager.MaxResponseTimeWidget.Text = utils.FormatDurationMillisecond(freshStats.MaxResponseTime)

		if freshStats.ChecksCount >= 2 {
			m.detailsManager.P95ResponseTimeWidget.Text = fmt.Sprintf("%d ms", freshStats.P95.Milliseconds())
		} else {
			m.detailsManager.P95ResponseTimeWidget.Text = "N/A"
		}
	}

	if data, exists := m.targetData[currentKey.String()]; exists {
		m.updateTargetDetails(data.Result, data.Stats)
	}

	if m.IsLogsVisible() {
		m.updateLogsWidget(currentKey)
	}
}

func (m *Manager) updateTargetDetails(result net.WebsiteCheckResult, _ stats.Stats) {
	sslExpiry := m.getSSLExpiry(result.URL)
	if sslExpiry > 0 {
		m.detailsManager.SSLOkWidget.Text = fmt.Sprintf("%d days remaining", sslExpiry)
	} else {
		m.detailsManager.SSLOkWidget.Text = "Checking..."
	}

	switch {
	case result.AssertText == "":
		m.detailsManager.AssertionWidget.Text = "N/A"
	case result.AssertionPassed:
		m.detailsManager.AssertionWidget.Text = "Passing"
	default:
		m.detailsManager.AssertionWidget.Text = "Failing"
	}

	if result.TraceInfo != nil {
		m.detailsManager.TimingBreakdownWidget.SetTimings(map[string]time.Duration{
			"Wait":     result.TraceInfo.Wait,
			"DNS":      result.TraceInfo.DNSLookup,
			"TCP":      result.TraceInfo.TCPConnection,
			"TTFB":     result.TraceInfo.TimeToFirstByte,
			"Download": result.TraceInfo.DownloadDuration,
		})
	}
}
