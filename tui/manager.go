package tui

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
	"github.com/Owloops/updo/utils"
	uw "github.com/Owloops/updo/widgets"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

const (
	notAvailable = "N/A"
	statusIcon   = "●"
	checking     = "Checking..."
	passing      = "Passing"
	failing      = "Failing"
)

type Manager struct {
	targets         []config.Target
	keyRegistry     *TargetKeyRegistry
	targetData      map[string]TargetData
	plotData        map[string]PlotHistory
	logBuffer       *LogBuffer
	sslExpiry       map[string]int
	sslExpiryMu     sync.RWMutex
	currentKeyIndex int
	isSingle        bool
	listWidget      *uw.FilteredList
	searchWidget    *widgets.Paragraph
	detailsManager  *DetailsManager
	grid            *ui.Grid
	termWidth       int
	termHeight      int
	focusOnLogs     bool
	showLogs        bool

	itemToKeyIndex          []int
	preserveHeaderSelection string
}

type PlotHistory struct {
	UptimeData       []float64
	ResponseTimeData []float64
}

func NewManager(targets []config.Target, options Options) *Manager {
	keyRegistry := NewTargetKeyRegistry(targets, options.Regions)
	allKeys := keyRegistry.GetAllKeys()

	m := &Manager{
		targets:         targets,
		keyRegistry:     keyRegistry,
		targetData:      make(map[string]TargetData),
		plotData:        make(map[string]PlotHistory),
		logBuffer:       NewLogBuffer(1000),
		sslExpiry:       make(map[string]int),
		currentKeyIndex: 0,
		isSingle:        len(allKeys) == 1,
		detailsManager:  NewDetailsManager(),
	}

	m.startSSLCollection()
	return m
}

func (m *Manager) startSSLCollection() {
	for _, target := range m.targets {
		go func(url string) {
			if strings.HasPrefix(url, "https://") {
				sslDaysRemaining := net.GetSSLCertExpiry(url)
				m.sslExpiryMu.Lock()
				m.sslExpiry[url] = sslDaysRemaining
				m.sslExpiryMu.Unlock()
			}
		}(target.URL)
	}
}

func (m *Manager) getSSLExpiry(url string) int {
	m.sslExpiryMu.RLock()
	defer m.sslExpiryMu.RUnlock()
	if days, exists := m.sslExpiry[url]; exists {
		return days
	}
	return 0
}

func (m *Manager) InitializeLayout(width, height int) {
	m.termWidth = width
	m.termHeight = height

	allKeys := m.keyRegistry.GetAllKeys()
	if len(allKeys) > 0 {
		firstKey := allKeys[0]
		var firstTarget config.Target
		for _, target := range m.targets {
			if target.Name == firstKey.TargetName {
				firstTarget = target
				break
			}
		}
		m.detailsManager.InitializeWidgets(firstTarget.URL, firstTarget.GetRefreshInterval())
	}

	if !m.isSingle {
		m.initializeMultiTargetWidgets()
	}

	m.setupGrid(width, height)
}

func (m *Manager) updateTargetList() {
	allKeys := m.keyRegistry.GetAllKeys()

	targetGroups := make(map[string][]TargetKey)
	targetOrder := make([]string, 0)

	for _, key := range allKeys {
		if _, exists := targetGroups[key.TargetName]; !exists {
			targetOrder = append(targetOrder, key.TargetName)
		}
		targetGroups[key.TargetName] = append(targetGroups[key.TargetName], key)
	}

	preserveSelection := false
	currentSelectedRow := -1
	preserveGroupID := m.preserveHeaderSelection

	if preserveGroupID == "" && m.listWidget != nil && len(m.itemToKeyIndex) > 0 {
		currentSelectedRow = m.listWidget.SelectedRow
		if currentSelectedRow >= 0 && currentSelectedRow < len(m.itemToKeyIndex) {
			if m.itemToKeyIndex[currentSelectedRow] == -1 {
				preserveSelection = true
			}
		}
	}

	items := make([]string, 0)
	metadata := make([]uw.RowMetadata, 0)

	itemToKeyIndex := make([]int, 0)

	keyIndex := 0
	for _, targetName := range targetOrder {
		keys := targetGroups[targetName]
		groupID := targetName

		isCollapsed := m.listWidget != nil && m.listWidget.IsGroupCollapsed(groupID)
		collapseIcon := "▼"
		if isCollapsed {
			collapseIcon = "▶"
		}

		header := fmt.Sprintf("%s %s", collapseIcon, targetName)
		if isCollapsed && len(keys) > 1 {
			header = fmt.Sprintf("%s %s (%d)", collapseIcon, targetName, len(keys))
		}
		items = append(items, header)
		metadata = append(metadata, uw.RowMetadata{
			GroupID:      groupID,
			IsHeader:     true,
			IsSelectable: true,
		})
		itemToKeyIndex = append(itemToKeyIndex, -1)

		if !isCollapsed {
			for _, key := range keys {

				var icon, iconColor string
				var responseTime string

				if data, exists := m.targetData[key.String()]; exists {
					if data.Result.IsUp {
						icon = "◉"
						iconColor = "green"
					} else {
						icon = "◉"
						iconColor = "red"
					}
					ms := int(data.Result.ResponseTime.Seconds() * 1000)
					responseTime = fmt.Sprintf("%dms", ms)
				} else {
					icon = "◉"
					iconColor = "yellow"
					responseTime = "..."
				}

				region := "local"
				if !key.IsLocal && key.Region != "" {
					region = key.Region
				}

				coloredIcon := fmt.Sprintf("[%s](fg:%s)", icon, iconColor)
				line := fmt.Sprintf("  %s %-10s %8s", coloredIcon, region, responseTime)

				isOnHeader := false
				if m.preserveHeaderSelection != "" {
					isOnHeader = true
				} else if m.listWidget != nil && m.listWidget.SelectedRow >= 0 {
					if m.listWidget.IsSearchMode() {
						if m.listWidget.IsHeaderAtIndex(m.listWidget.SelectedRow) {
							isOnHeader = true
						}
					} else if m.listWidget.SelectedRow < len(m.itemToKeyIndex) {
						if m.itemToKeyIndex[m.listWidget.SelectedRow] == -1 {
							isOnHeader = true
						}
					}
				}

				if keyIndex == m.currentKeyIndex && !isOnHeader {
					items = append(items, "▶ "+line)
				} else {
					items = append(items, "  "+line)
				}

				metadata = append(metadata, uw.RowMetadata{
					GroupID:      groupID,
					IsHeader:     false,
					IsSelectable: true,
				})

				itemToKeyIndex = append(itemToKeyIndex, keyIndex)

				keyIndex++
			}
		} else {
			keyIndex += len(keys)
		}
	}

	m.itemToKeyIndex = itemToKeyIndex

	m.listWidget.SetRowsWithMetadata(items, metadata)

	currentItemIndex := -1
	for i, keyIdx := range m.itemToKeyIndex {
		if keyIdx == m.currentKeyIndex {
			currentItemIndex = i
			break
		}
	}

	if preserveGroupID != "" {
		if m.listWidget.IsSearchMode() {
			filteredIndices := m.listWidget.GetFilteredIndices()
			for displayIdx, originalIdx := range filteredIndices {
				if originalIdx < len(metadata) && metadata[originalIdx].IsHeader && metadata[originalIdx].GroupID == preserveGroupID {
					m.listWidget.SelectedRow = displayIdx
					break
				}
			}
		} else {
			for i := range items {
				if i < len(metadata) && metadata[i].IsHeader && metadata[i].GroupID == preserveGroupID {
					m.listWidget.SelectedRow = i
					break
				}
			}
		}
	} else if preserveSelection && currentSelectedRow >= 0 && currentSelectedRow < len(items) {
		m.listWidget.SelectedRow = currentSelectedRow
	} else if currentItemIndex >= 0 {
		if m.listWidget.IsSearchMode() {
			filteredIndices := m.listWidget.GetFilteredIndices()
			for displayIdx, originalIdx := range filteredIndices {
				if originalIdx == currentItemIndex {
					m.listWidget.SelectedRow = displayIdx
					break
				}
			}
		} else {
			m.listWidget.SelectedRow = currentItemIndex
		}
	}

	keyVisible := true
	if m.listWidget.IsSearchMode() {
		indices := m.listWidget.GetFilteredIndices()
		keyVisible = false
		for i, keyIdx := range m.itemToKeyIndex {
			if keyIdx == m.currentKeyIndex && slices.Contains(indices, i) {
				keyVisible = true
				break
			}
		}
	}

	isOnHeader := false
	if m.listWidget.SelectedRow >= 0 {
		if m.listWidget.IsSearchMode() {
			if m.listWidget.IsHeaderAtIndex(m.listWidget.SelectedRow) {
				isOnHeader = true
			}
		} else if m.listWidget.SelectedRow < len(m.itemToKeyIndex) {
			if m.itemToKeyIndex[m.listWidget.SelectedRow] == -1 {
				isOnHeader = true
			}
		}
	}

	if isOnHeader {
		m.listWidget.SelectedRowStyle.Fg = ui.ColorMagenta
		m.listWidget.SelectedRowStyle.Modifier = ui.ModifierBold
	} else if keyVisible && m.currentKeyIndex >= 0 && m.currentKeyIndex < len(allKeys) {
		currentKey := allKeys[m.currentKeyIndex]
		if data, exists := m.targetData[currentKey.String()]; exists {
			if data.Result.IsUp {
				m.listWidget.SelectedRowStyle.Fg = ui.ColorGreen
			} else {
				m.listWidget.SelectedRowStyle.Fg = ui.ColorRed
			}
		} else {
			m.listWidget.SelectedRowStyle.Fg = ui.ColorYellow
		}
		m.listWidget.SelectedRowStyle.Modifier = ui.ModifierBold
	} else {
		m.listWidget.SelectedRowStyle.Fg = ui.ColorCyan
		m.listWidget.SelectedRowStyle.Modifier = ui.ModifierBold
	}
}

func (m *Manager) SetActiveTargetKey(keyIndex int, monitors map[string]*stats.Monitor) {
	allKeys := m.keyRegistry.GetAllKeys()
	if keyIndex >= 0 && keyIndex < len(allKeys) {
		m.currentKeyIndex = keyIndex
		m.updateActiveTarget(monitors)
	}
}

func (m *Manager) updateActiveTarget(_ map[string]*stats.Monitor) {
	allKeys := m.keyRegistry.GetAllKeys()
	if m.currentKeyIndex < 0 || m.currentKeyIndex >= len(allKeys) {
		return
	}

	currentKey := allKeys[m.currentKeyIndex]
	targetKeyStr := currentKey.String()

	m.restorePlotData(targetKeyStr)

	if data, exists := m.targetData[targetKeyStr]; exists {
		m.updateCurrentTargetWidgets(data.Result, data.Stats)
	}

	if m.showLogs {
		m.updateLogsWidget(currentKey)
	}

	if !m.isSingle {
		m.updateTargetList()
	}

	ui.Render(m.grid)
}

func (m *Manager) UpdateTarget(data TargetData) {
	targetKeyStr := data.TargetKey.String()
	m.targetData[targetKeyStr] = data

	m.updatePlotDataForTarget(targetKeyStr, data.Result)

	if data.WebhookError != nil {
		m.logBuffer.AddLogEntry(LogLevelWarning, "Webhook failed", data.WebhookError.Error(), data.TargetKey)
	}

	if !data.Result.IsUp {
		level := LogLevelError
		message := "Request failed"

		if data.Result.StatusCode > 0 {
			message = fmt.Sprintf("Status code: %d", data.Result.StatusCode)
		} else if !data.TargetKey.IsLocal {
			message = "Lambda invocation failed"
			level = LogLevelWarning
		}

		m.logBuffer.AddLogEntry(level, message, "", data.TargetKey)
	} else if m.logBuffer.Size() == 0 || m.logBuffer.Size()%10 == 0 {
		m.logBuffer.AddLogEntry(LogLevelInfo, "Request successful", "", data.TargetKey)
	}

	allKeys := m.keyRegistry.GetAllKeys()
	if m.currentKeyIndex >= 0 && m.currentKeyIndex < len(allKeys) {
		currentKey := allKeys[m.currentKeyIndex]
		if currentKey.String() == targetKeyStr {
			m.restorePlotData(targetKeyStr)
			m.updateCurrentTargetWidgets(data.Result, data.Stats)
			if m.showLogs {
				m.updateLogsWidget(currentKey)
			}
			ui.Render(m.grid)
		} else if !m.isSingle {
			m.updateTargetList()
			ui.Render(m.grid)
		}
	}
}

func (m *Manager) RefreshStats(monitors map[string]*stats.Monitor) {
	allKeys := m.keyRegistry.GetAllKeys()
	if m.currentKeyIndex < 0 || m.currentKeyIndex >= len(allKeys) {
		return
	}

	currentKey := allKeys[m.currentKeyIndex]
	if monitor, exists := monitors[currentKey.String()]; exists {
		freshStats := monitor.GetStats()

		m.detailsManager.UptimeWidget.Text = fmt.Sprintf("%.2f%%", freshStats.UptimePercent)
		m.detailsManager.UpForWidget.Text = utils.FormatDurationMinute(freshStats.TotalDuration)

		if freshStats.ChecksCount > 0 && freshStats.SuccessCount > 0 {
			m.detailsManager.AvgResponseTimeWidget.Text = utils.FormatDurationMillisecond(freshStats.AvgResponseTime)
			m.detailsManager.MinResponseTimeWidget.Text = utils.FormatDurationMillisecond(freshStats.MinResponseTime)
			m.detailsManager.MaxResponseTimeWidget.Text = utils.FormatDurationMillisecond(freshStats.MaxResponseTime)
		} else {
			m.detailsManager.AvgResponseTimeWidget.Text = notAvailable
			m.detailsManager.MinResponseTimeWidget.Text = notAvailable
			m.detailsManager.MaxResponseTimeWidget.Text = notAvailable
		}

		if freshStats.ChecksCount >= 2 {
			m.detailsManager.P95ResponseTimeWidget.Text = fmt.Sprintf("%d ms", freshStats.P95.Milliseconds())
		} else {
			m.detailsManager.P95ResponseTimeWidget.Text = notAvailable
		}

		if !m.isSingle {
			isOnHeader := false
			if m.listWidget != nil && m.listWidget.SelectedRow >= 0 {
				if m.listWidget.IsSearchMode() {
					if m.listWidget.IsHeaderAtIndex(m.listWidget.SelectedRow) {
						isOnHeader = true
					}
				} else if m.listWidget.SelectedRow < len(m.itemToKeyIndex) {
					if m.itemToKeyIndex[m.listWidget.SelectedRow] == -1 {
						isOnHeader = true
					}
				}
			}
			if !isOnHeader {
				m.updateTargetList()
			}
		}
		ui.Render(m.grid)
	}
}

func (m *Manager) updateCurrentTargetWidgets(result net.WebsiteCheckResult, stats stats.Stats) {
	m.detailsManager.UptimeWidget.Text = fmt.Sprintf("%.2f%%", stats.UptimePercent)
	m.detailsManager.UpForWidget.Text = utils.FormatDurationMinute(stats.TotalDuration)

	if stats.ChecksCount > 0 && stats.SuccessCount > 0 {
		m.detailsManager.AvgResponseTimeWidget.Text = utils.FormatDurationMillisecond(stats.AvgResponseTime)
		m.detailsManager.MinResponseTimeWidget.Text = utils.FormatDurationMillisecond(stats.MinResponseTime)
		m.detailsManager.MaxResponseTimeWidget.Text = utils.FormatDurationMillisecond(stats.MaxResponseTime)
	} else {
		m.detailsManager.AvgResponseTimeWidget.Text = notAvailable
		m.detailsManager.MinResponseTimeWidget.Text = notAvailable
		m.detailsManager.MaxResponseTimeWidget.Text = notAvailable
	}

	if stats.ChecksCount >= 2 {
		m.detailsManager.P95ResponseTimeWidget.Text = fmt.Sprintf("%d ms", stats.P95.Milliseconds())
	} else {
		m.detailsManager.P95ResponseTimeWidget.Text = notAvailable
	}

	sslExpiry := m.getSSLExpiry(result.URL)
	if sslExpiry > 0 {
		m.detailsManager.SSLOkWidget.Text = fmt.Sprintf("%d days remaining", sslExpiry)
	} else {
		m.detailsManager.SSLOkWidget.Text = checking
	}

	switch {
	case result.AssertText == "":
		m.detailsManager.AssertionWidget.Text = notAvailable
	case result.AssertionPassed:
		m.detailsManager.AssertionWidget.Text = passing
	default:
		m.detailsManager.AssertionWidget.Text = failing
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

	m.detailsManager.updatePlotsData(result, m.termWidth)
}

func (m *Manager) restorePlotData(targetName string) {
	if history, exists := m.plotData[targetName]; exists {
		m.detailsManager.UptimePlot.Data[0] = slices.Clone(history.UptimeData)
		m.detailsManager.ResponseTimePlot.Data[0] = slices.Clone(history.ResponseTimeData)
	} else {
		m.detailsManager.UptimePlot.Data[0] = make([]float64, 0)
		m.detailsManager.ResponseTimePlot.Data[0] = []float64{0.0, 0.0}
	}
}

func (m *Manager) updatePlotDataForTarget(targetName string, result net.WebsiteCheckResult) {
	history, exists := m.plotData[targetName]
	if !exists {
		history = PlotHistory{
			UptimeData:       make([]float64, 0),
			ResponseTimeData: []float64{0.0, 0.0},
		}
	}

	history.UptimeData = append(history.UptimeData, utils.BoolToFloat64(result.IsUp))
	history.ResponseTimeData = append(history.ResponseTimeData, result.ResponseTime.Seconds())

	maxLength := m.termWidth / 2

	if len(history.UptimeData) > maxLength {
		history.UptimeData = history.UptimeData[len(history.UptimeData)-maxLength:]
	}

	if len(history.ResponseTimeData) > maxLength {
		history.ResponseTimeData = history.ResponseTimeData[len(history.ResponseTimeData)-maxLength:]
	}

	m.plotData[targetName] = history
}
