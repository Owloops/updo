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
	_notAvailable          = "N/A"
	_checking              = "Checking..."
	_passing               = "Passing"
	_failing               = "Failing"
	_targetIcon            = "◉"
	_backspaceKey          = "<Backspace>"
	_ctrlBackspace         = "<C-8>"
	_defaultRefresh        = 5
	_logBufferSize         = 1000
	_dataChannelMultiplier = 2
	_targetsTitle          = "Targets"
)

type Manager struct {
	targets         []config.Target
	keyRegistry     *stats.TargetKeyRegistry
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
	keyRegistry := stats.NewTargetKeyRegistry(targets, options.Regions)
	allKeys := keyRegistry.GetAllKeys()

	m := &Manager{
		targets:         targets,
		keyRegistry:     keyRegistry,
		targetData:      make(map[string]TargetData, len(allKeys)),
		plotData:        make(map[string]PlotHistory, len(allKeys)),
		logBuffer:       NewLogBuffer(_logBufferSize),
		sslExpiry:       make(map[string]int, len(targets)),
		currentKeyIndex: 0,
		isSingle:        len(allKeys) == 1,
		detailsManager:  NewDetailsManager(),
	}

	return m
}

func (m *Manager) getSSLExpiry(url string) int {
	m.sslExpiryMu.RLock()
	if days, exists := m.sslExpiry[url]; exists {
		m.sslExpiryMu.RUnlock()
		return days
	}
	m.sslExpiryMu.RUnlock()

	if strings.HasPrefix(url, "https://") {
		go func(sslURL string) {
			sslDaysRemaining := net.GetSSLCertExpiry(sslURL)
			m.sslExpiryMu.Lock()
			m.sslExpiry[sslURL] = sslDaysRemaining
			m.sslExpiryMu.Unlock()
		}(url)
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
		if m.isSingle && len(m.targets) > 0 {
			m.detailsManager.InitializeWidgets(m.targets[0].URL, m.targets[0].GetRefreshInterval())
		} else {
			m.detailsManager.InitializeWidgets(firstTarget.URL, firstTarget.GetRefreshInterval())
		}
	}

	if !m.isSingle {
		m.initializeMultiTargetWidgets()
	}

	m.setupGrid(width, height)
}

func (m *Manager) updateTargetList() {
	if m.listWidget == nil {
		return
	}
	allKeys := m.keyRegistry.GetAllKeys()

	targetGroups := make(map[string][]stats.TargetKey, len(allKeys))
	var targetOrder []string

	for _, key := range allKeys {
		displayName := key.GetCleanName()

		if _, exists := targetGroups[displayName]; !exists {
			targetOrder = append(targetOrder, displayName)
		}
		targetGroups[displayName] = append(targetGroups[displayName], key)
	}

	preserveGroupID := m.preserveHeaderSelection

	var items []string
	var metadata []uw.RowMetadata
	var itemToKeyIndex []int

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

				if data, exists := m.targetData[key.String()]; exists {
					if data.Result.IsUp {
						icon = _targetIcon
						iconColor = "green"
					} else {
						icon = _targetIcon
						iconColor = "red"
					}
				} else {
					icon = _targetIcon
					iconColor = "yellow"
				}

				region := "local"
				if !key.IsLocal && key.Region != "" {
					region = key.Region
				}

				coloredIcon := fmt.Sprintf("[%s](fg:%s)", icon, iconColor)
				line := fmt.Sprintf("  %s %s", coloredIcon, region)

				items = append(items, "  "+line)

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
	}

	if m.listWidget.SelectedRow >= len(items) || m.listWidget.SelectedRow < 0 {
		m.listWidget.SelectedRow = 0
	}

	m.updateSelectionColors()
}

func (m *Manager) getCurrentTargetKey() *stats.TargetKey {
	if m.isSingle || m.listWidget == nil {
		allKeys := m.keyRegistry.GetAllKeys()
		if m.currentKeyIndex >= 0 && m.currentKeyIndex < len(allKeys) {
			return &allKeys[m.currentKeyIndex]
		}
		return nil
	}

	selectedRow := m.listWidget.SelectedRow
	if selectedRow < 0 {
		return nil
	}

	var originalIdx int
	if m.listWidget.IsSearchMode() {
		filteredIndices := m.listWidget.GetFilteredIndices()
		if selectedRow >= len(filteredIndices) {
			return nil
		}
		originalIdx = filteredIndices[selectedRow]
	} else {
		originalIdx = selectedRow
	}

	if originalIdx >= 0 && originalIdx < len(m.itemToKeyIndex) {
		keyIdx := m.itemToKeyIndex[originalIdx]
		if keyIdx >= 0 {
			allKeys := m.keyRegistry.GetAllKeys()
			if keyIdx < len(allKeys) {
				return &allKeys[keyIdx]
			}
		}
	}
	return nil
}

func (m *Manager) getKeysForCurrentSelection() []stats.TargetKey {
	if m.isSingle || m.listWidget == nil {
		allKeys := m.keyRegistry.GetAllKeys()
		if m.currentKeyIndex >= 0 && m.currentKeyIndex < len(allKeys) {
			return []stats.TargetKey{allKeys[m.currentKeyIndex]}
		}
		return nil
	}

	selectedRow := m.listWidget.SelectedRow
	if selectedRow < 0 {
		return nil
	}

	if m.listWidget.IsHeaderAtIndex(selectedRow) {
		groupID := m.listWidget.GetGroupAtIndex(selectedRow)
		if groupID == "" {
			return nil
		}

		allKeys := m.keyRegistry.GetAllKeys()
		targetGroups := make(map[string][]stats.TargetKey)

		for _, key := range allKeys {
			displayName := key.GetCleanName()
			targetGroups[displayName] = append(targetGroups[displayName], key)
		}

		if keys, exists := targetGroups[groupID]; exists {
			return keys
		}
		return nil
	}

	if key := m.getCurrentTargetKey(); key != nil {
		return []stats.TargetKey{*key}
	}

	return nil
}

func (m *Manager) getCurrentTarget() *config.Target {
	currentKey := m.getCurrentTargetKey()
	if currentKey == nil {
		return nil
	}

	if currentKey.TargetIndex >= 0 && currentKey.TargetIndex < len(m.targets) {
		return &m.targets[currentKey.TargetIndex]
	}
	return nil
}

func (m *Manager) isSelectedRowHeader() bool {
	if m.isSingle || m.listWidget == nil {
		return false
	}

	selectedRow := m.listWidget.SelectedRow
	if selectedRow < 0 || selectedRow >= len(m.listWidget.Rows) {
		return false
	}

	row := m.listWidget.Rows[selectedRow]
	return strings.HasPrefix(row, "▼") || strings.HasPrefix(row, "▶")
}

func (m *Manager) updateSelectionColors() {
	if m.isSingle || m.listWidget == nil {
		return
	}

	if m.isSelectedRowHeader() {
		m.listWidget.SelectedRowStyle.Fg = ui.ColorMagenta
		m.listWidget.SelectedRowStyle.Modifier = ui.ModifierBold
		return
	}
	currentKey := m.getCurrentTargetKey()
	if currentKey != nil {
		if data, exists := m.targetData[currentKey.String()]; exists {
			if data.Result.IsUp {
				m.listWidget.SelectedRowStyle.Fg = ui.ColorGreen
			} else {
				m.listWidget.SelectedRowStyle.Fg = ui.ColorRed
			}
		} else {
			m.listWidget.SelectedRowStyle.Fg = ui.ColorYellow
		}
	} else {
		m.listWidget.SelectedRowStyle.Fg = ui.ColorCyan
	}
	m.listWidget.SelectedRowStyle.Modifier = ui.ModifierBold
}

func (m *Manager) SetActiveTargetKey(keyIndex int, monitors map[string]*stats.Monitor) {
	allKeys := m.keyRegistry.GetAllKeys()
	if keyIndex >= 0 && keyIndex < len(allKeys) {
		m.currentKeyIndex = keyIndex
		m.updateActiveTarget(monitors)
	}
}

func (m *Manager) updateActiveTarget(monitors map[string]*stats.Monitor) {
	currentKey := m.getCurrentTargetKey()
	if currentKey == nil {
		return
	}

	currentTarget := m.getCurrentTarget()
	if currentTarget != nil && m.detailsManager.RefreshWidget != nil {
		refreshInterval := currentTarget.RefreshInterval
		if refreshInterval == 0 {
			refreshInterval = _defaultRefresh
		}
		m.detailsManager.RefreshWidget.Text = fmt.Sprintf("%d seconds", refreshInterval)
	}

	if currentTarget != nil && m.detailsManager.URLWidget != nil {
		m.detailsManager.URLWidget.Text = currentTarget.URL
	}

	targetKeyStr := currentKey.String()

	m.restorePlotData(targetKeyStr)

	if monitor, exists := monitors[targetKeyStr]; exists {
		freshStats := monitor.GetStats()
		if data, exists := m.targetData[targetKeyStr]; exists {
			m.updateCurrentTargetWidgets(data.Result, freshStats)
		} else {
			m.detailsManager.UpForWidget.Text = utils.FormatDurationMinute(freshStats.TotalDuration)
			m.detailsManager.UptimeWidget.Text = fmt.Sprintf("%.2f%%", freshStats.UptimePercent)
		}
	} else if data, exists := m.targetData[targetKeyStr]; exists {
		m.updateCurrentTargetWidgets(data.Result, data.Stats)
	}

	if m.showLogs {
		m.updateLogsWidget(*currentKey)
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

	if data.LambdaError != nil {
		m.logBuffer.AddLogEntry(LogLevelWarning, "Lambda invocation failed", data.LambdaError.Error(), data.TargetKey)
	}

	if data.AlertError != nil {
		m.logBuffer.AddLogEntry(LogLevelWarning, "Alert notification failed", data.AlertError.Error(), data.TargetKey)
	}

	if !data.Result.IsUp {
		level := LogLevelError
		message := "Request failed"

		switch {
		case data.Result.AssertText != "" && !data.Result.AssertionPassed && data.Result.StatusCode >= 200 && data.Result.StatusCode < 300:
			message = fmt.Sprintf("Assertion failed (status %d)", data.Result.StatusCode)
			level = LogLevelWarning
		case data.Result.StatusCode > 0:
			message = fmt.Sprintf("Status code: %d", data.Result.StatusCode)
		case !data.TargetKey.IsLocal && data.LambdaError == nil:
			message = "Lambda invocation failed"
			level = LogLevelWarning
		}

		m.logBuffer.AddLogEntry(level, message, "", data.TargetKey)
	} else if m.logBuffer.Size() == 0 || m.logBuffer.Size()%10 == 0 {
		m.logBuffer.AddLogEntry(LogLevelInfo, "Request successful", "", data.TargetKey)
	}

	currentKey := m.getCurrentTargetKey()
	if currentKey != nil && currentKey.String() == targetKeyStr {
		if m.isSingle && m.detailsManager.URLWidget != nil {
			m.detailsManager.URLWidget.Text = data.Target.URL
		}
		m.restorePlotData(targetKeyStr)
		m.updateCurrentTargetWidgets(data.Result, data.Stats)
		if m.showLogs {
			m.updateLogsWidget(*currentKey)
		}
		ui.Render(m.grid)
	} else if !m.isSingle {
		m.updateTargetList()
		ui.Render(m.grid)
	}
}

func (m *Manager) RefreshStats(monitors map[string]*stats.Monitor) {
	currentKey := m.getCurrentTargetKey()
	if currentKey == nil {
		return
	}

	currentTarget := m.getCurrentTarget()
	if currentTarget != nil && m.detailsManager.RefreshWidget != nil {
		refreshInterval := currentTarget.RefreshInterval
		if refreshInterval == 0 {
			refreshInterval = _defaultRefresh
		}
		m.detailsManager.RefreshWidget.Text = fmt.Sprintf("%d seconds", refreshInterval)
	}

	if monitor, exists := monitors[currentKey.String()]; exists {
		freshStats := monitor.GetStats()

		m.detailsManager.UptimeWidget.Text = fmt.Sprintf("%.2f%%", freshStats.UptimePercent)
		m.detailsManager.UpForWidget.Text = utils.FormatDurationMinute(freshStats.TotalDuration)

		if freshStats.ChecksCount > 0 {
			m.detailsManager.AvgResponseTimeWidget.Text = utils.FormatDurationMillisecond(freshStats.AvgResponseTime)
			m.detailsManager.MinResponseTimeWidget.Text = utils.FormatDurationMillisecond(freshStats.MinResponseTime)
			m.detailsManager.MaxResponseTimeWidget.Text = utils.FormatDurationMillisecond(freshStats.MaxResponseTime)
		} else {
			m.detailsManager.AvgResponseTimeWidget.Text = _notAvailable
			m.detailsManager.MinResponseTimeWidget.Text = _notAvailable
			m.detailsManager.MaxResponseTimeWidget.Text = _notAvailable
		}

		if freshStats.ChecksCount >= 2 {
			m.detailsManager.P95ResponseTimeWidget.Text = fmt.Sprintf("%d ms", freshStats.P95.Milliseconds())
		} else {
			m.detailsManager.P95ResponseTimeWidget.Text = _notAvailable
		}

		if !m.isSingle {
			if !m.isSelectedRowHeader() {
				m.updateTargetList()
			}
		}
		ui.Render(m.grid)
	}
}

func (m *Manager) updateCurrentTargetWidgets(result net.WebsiteCheckResult, stats stats.Stats) {
	m.detailsManager.UptimeWidget.Text = fmt.Sprintf("%.2f%%", stats.UptimePercent)
	m.detailsManager.UpForWidget.Text = utils.FormatDurationMinute(stats.TotalDuration)

	if stats.ChecksCount > 0 {
		m.detailsManager.AvgResponseTimeWidget.Text = utils.FormatDurationMillisecond(stats.AvgResponseTime)
		m.detailsManager.MinResponseTimeWidget.Text = utils.FormatDurationMillisecond(stats.MinResponseTime)
		m.detailsManager.MaxResponseTimeWidget.Text = utils.FormatDurationMillisecond(stats.MaxResponseTime)
	} else {
		m.detailsManager.AvgResponseTimeWidget.Text = _notAvailable
		m.detailsManager.MinResponseTimeWidget.Text = _notAvailable
		m.detailsManager.MaxResponseTimeWidget.Text = _notAvailable
	}

	if stats.ChecksCount >= 2 {
		m.detailsManager.P95ResponseTimeWidget.Text = fmt.Sprintf("%d ms", stats.P95.Milliseconds())
	} else {
		m.detailsManager.P95ResponseTimeWidget.Text = _notAvailable
	}

	sslExpiry := m.getSSLExpiry(result.URL)
	if sslExpiry > 0 {
		m.detailsManager.SSLOkWidget.Text = fmt.Sprintf("%d days remaining", sslExpiry)
	} else {
		m.detailsManager.SSLOkWidget.Text = _checking
	}

	switch {
	case result.AssertText == "":
		m.detailsManager.AssertionWidget.Text = _notAvailable
	case result.AssertionPassed:
		m.detailsManager.AssertionWidget.Text = _passing
	default:
		m.detailsManager.AssertionWidget.Text = _failing
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
		m.detailsManager.UptimePlot.Data[0] = nil
		m.detailsManager.ResponseTimePlot.Data[0] = []float64{0.0, 0.0}
	}
}

func (m *Manager) updatePlotDataForTarget(targetName string, result net.WebsiteCheckResult) {
	history, exists := m.plotData[targetName]
	if !exists {
		history = PlotHistory{
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
