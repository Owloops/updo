package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
	"github.com/Owloops/updo/utils"
	uw "github.com/Owloops/updo/widgets"

	ui "github.com/Owloops/termui/v3"
)

const (
	_targetIcon            = "◉"
	_backspaceKey          = "<Backspace>"
	_ctrlBackspace         = "<C-8>"
	_logBufferSize         = 1000
	_dataChannelMultiplier = 2
	_sparkWidth            = 8
)

type Manager struct {
	targets         []config.Target
	options         Options
	keyRegistry     *stats.TargetKeyRegistry
	targetData      map[string]TargetData
	logBuffer       *LogBuffer
	sslExpiry       map[string]int
	sslExpiryMu     sync.RWMutex
	currentKeyIndex int
	isSingle        bool
	listWidget      *uw.FilteredList
	searchWidget    *uw.SearchBox
	detailsManager  *DetailsManager
	grid            ui.Drawable
	termWidth       int
	termHeight      int
	focusOnLogs     bool
	showLogs        bool

	itemToKeyIndex          []int
	preserveHeaderSelection string

	pal       palette
	dash      *dashWidgets
	history   map[string][]uw.Sample
	liveStats map[string]stats.Stats
}

func NewManager(targets []config.Target, options Options) *Manager {
	keyRegistry := stats.NewTargetKeyRegistry(targets, options.Regions)
	allKeys := keyRegistry.GetAllKeys()

	m := &Manager{
		targets:         targets,
		options:         options,
		keyRegistry:     keyRegistry,
		targetData:      make(map[string]TargetData, len(allKeys)),
		logBuffer:       NewLogBuffer(_logBufferSize),
		sslExpiry:       make(map[string]int, len(targets)),
		currentKeyIndex: 0,
		isSingle:        len(allKeys) == 1,
		detailsManager:  NewDetailsManager(),
		pal:             newPalette(accentFromEnv()),
		history:         make(map[string][]uw.Sample, len(allKeys)),
		liveStats:       make(map[string]stats.Stats, len(allKeys)),
	}
	return m
}

func (m *Manager) getSSLExpiry(url string) int {
	m.sslExpiryMu.Lock()
	days, exists := m.sslExpiry[url]
	if !exists {
		m.sslExpiry[url] = 0
	}
	m.sslExpiryMu.Unlock()
	if exists {
		return days
	}

	if strings.HasPrefix(url, "https://") {
		go func(sslURL string) {
			sslDaysRemaining := net.GetSSLCertExpiry(sslURL, m.options.IPVersion)
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

	m.detailsManager.InitializeWidgets()

	if !m.isSingle {
		m.initializeMultiTargetWidgets()
	}

	m.initStyledWidgets()
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

		if len(keys) == 1 && keys[0].IsLocal {
			items = append(items, m.targetRow(keys[0], targetName))
			metadata = append(metadata, m.targetRowMetadata(keys[0], "", targetName))
			itemToKeyIndex = append(itemToKeyIndex, keyIndex)
			keyIndex++
			continue
		}

		isCollapsed := m.listWidget.IsGroupCollapsed(groupID)
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
			Label:        targetName,
		})
		itemToKeyIndex = append(itemToKeyIndex, -1)

		if isCollapsed {
			keyIndex += len(keys)
			continue
		}
		for _, key := range keys {
			region := _localRegion
			if !key.IsLocal && key.Region != "" {
				region = key.Region
			}
			items = append(items, "    "+m.targetRow(key, region))
			metadata = append(metadata, m.targetRowMetadata(key, groupID, region))
			itemToKeyIndex = append(itemToKeyIndex, keyIndex)
			keyIndex++
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
}

func (m *Manager) targetRow(key stats.TargetKey, label string) string {
	iconColor := "yellow"
	if data, exists := m.targetData[key.String()]; exists {
		iconColor = "red"
		if data.Result.IsUp {
			iconColor = "green"
		}
	}
	return fmt.Sprintf("[%s](fg:%s) %s", _targetIcon, iconColor, label)
}

func (m *Manager) targetRowMetadata(key stats.TargetKey, groupID, label string) uw.RowMetadata {
	right, rightStyle := m.rowLatency(key)
	return uw.RowMetadata{
		GroupID:      groupID,
		IsSelectable: true,
		Label:        label,
		Right:        right,
		RightStyle:   rightStyle,
		Spark:        uw.Sparkline(m.history[key.String()], _sparkWidth),
	}
}

func (m *Manager) rowLatency(key stats.TargetKey) (string, ui.Style) {
	data, exists := m.targetData[key.String()]
	switch {
	case !exists:
		return _placeholder, m.pal.muted
	case !data.Result.IsUp:
		return "down", ui.NewStyle(m.pal.bad.Fg)
	}
	return formatDuration(data.Result.ResponseTime), m.pal.muted
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

	targetKeyStr := currentKey.String()
	if monitor, exists := monitors[targetKeyStr]; exists {
		m.liveStats[targetKeyStr] = monitor.GetStats()
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
	if data.Stats.ChecksCount > 0 {
		m.liveStats[targetKeyStr] = data.Stats
	}

	m.rememberSample(targetKeyStr, data.Result)

	logAdded := false

	if data.WebhookError != nil {
		m.logBuffer.AddLogEntry(LogLevelWarning, "Webhook failed", data.WebhookError.Error(), data.TargetKey)
		logAdded = true
	}

	if data.LambdaError != nil {
		m.logBuffer.AddLogEntry(LogLevelWarning, "Lambda invocation failed", data.LambdaError.Error(), data.TargetKey)
		logAdded = true
	}

	if data.AlertError != nil {
		m.logBuffer.AddLogEntry(LogLevelWarning, "Alert notification failed", data.AlertError.Error(), data.TargetKey)
		logAdded = true
	}

	if data.Result.ResponseTruncated {
		m.logBuffer.AddLogEntry(LogLevelWarning, "Response body truncated", "Exceeded BodySizeLimit; assertion checks may be unreliable", data.TargetKey)
		logAdded = true
	}

	if !data.Result.IsUp && data.LambdaError == nil {
		level := LogLevelError
		message := "Request failed"

		switch {
		case data.Result.AssertText != "" && !data.Result.AssertionPassed && data.Result.StatusCode >= 200 && data.Result.StatusCode < 300:
			message = fmt.Sprintf("Assertion failed (status %d)", data.Result.StatusCode)
			level = LogLevelWarning
		case data.Result.StatusCode > 0:
			message = fmt.Sprintf("Status code: %d", data.Result.StatusCode)
		case !data.TargetKey.IsLocal:
			message = "Lambda invocation failed"
			level = LogLevelWarning
		}

		m.logBuffer.AddLogEntry(level, message, "", data.TargetKey)
		logAdded = true
	} else if data.Result.IsUp && (m.logBuffer.Size() == 0 || m.logBuffer.Size()%10 == 0) {
		m.logBuffer.AddLogEntry(LogLevelInfo, "Request successful", "", data.TargetKey)
		logAdded = true
	}

	currentKey := m.getCurrentTargetKey()
	if currentKey != nil && currentKey.String() == targetKeyStr {
		if m.showLogs && logAdded {
			keys := m.getKeysForCurrentSelection()
			if len(keys) > 0 {
				m.updateLogsWidgetForTargets(keys)
			}
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

	monitor, exists := monitors[currentKey.String()]
	if !exists {
		return
	}
	m.liveStats[currentKey.String()] = monitor.GetStats()

	if !m.isSingle && !m.isSelectedRowHeader() {
		m.updateTargetList()
	}
	ui.Render(m.grid)
}

func stageTimings(result net.WebsiteCheckResult) map[string]time.Duration {
	if result.TraceInfo == nil {
		return map[string]time.Duration{}
	}
	return map[string]time.Duration{
		uw.StageWait:     utils.SanitizeDuration(result.TraceInfo.Wait),
		uw.StageDNS:      utils.SanitizeDuration(result.TraceInfo.DNSLookup),
		uw.StageTCP:      utils.SanitizeDuration(result.TraceInfo.TCPConnection),
		uw.StageTLS:      utils.SanitizeDuration(result.TraceInfo.TLSHandshake),
		uw.StageTTFB:     utils.SanitizeDuration(result.TraceInfo.TimeToFirstByte),
		uw.StageDownload: utils.SanitizeDuration(result.TraceInfo.DownloadDuration),
	}
}
