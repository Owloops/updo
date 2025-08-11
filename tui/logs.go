package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Owloops/updo/stats"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type LogLevel string

const (
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
)

type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
	Details   string
	TargetKey stats.TargetKey
}

type LogBuffer struct {
	entries []LogEntry
	head    int
	size    int
	maxSize int
	full    bool
	mu      sync.RWMutex
}

const (
	_defaultLogBufferSize = 100
	_maxLogsToShow        = 10
)

func NewLogBuffer(maxSize int) *LogBuffer {
	if maxSize <= 0 {
		maxSize = _defaultLogBufferSize
	}
	return &LogBuffer{
		entries: make([]LogEntry, maxSize),
		maxSize: maxSize,
	}
}

func (lb *LogBuffer) Add(entry LogEntry) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.entries[lb.head] = entry
	lb.head = (lb.head + 1) % lb.maxSize

	if !lb.full {
		lb.size++
		if lb.size == lb.maxSize {
			lb.full = true
		}
	}
}

func (lb *LogBuffer) GetEntries() []LogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if lb.size == 0 {
		return nil
	}

	result := make([]LogEntry, lb.size)

	if lb.full {
		copy(result, lb.entries[lb.head:])
		copy(result[lb.maxSize-lb.head:], lb.entries[:lb.head])
	} else {
		copy(result, lb.entries[:lb.size])
	}

	return result
}

func (lb *LogBuffer) GetRecentEntries(n int) []LogEntry {
	entries := lb.GetEntries()

	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	if n > 0 && n < len(entries) {
		return entries[:n]
	}
	return entries
}

func (lb *LogBuffer) Size() int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.size
}

func (lb *LogBuffer) Clear() {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.head = 0
	lb.size = 0
	lb.full = false
}

func (lb *LogBuffer) IsEmpty() bool {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.size == 0
}

func (lb *LogBuffer) MaxSize() int {
	return lb.maxSize
}

func (lb *LogBuffer) GetEntriesForTarget(targetKey stats.TargetKey) []LogEntry {
	entries := lb.GetEntries()
	var filtered []LogEntry

	for _, entry := range entries {
		if entry.TargetKey.String() == targetKey.String() {
			filtered = append(filtered, entry)
		}
	}

	return filtered
}

func (lb *LogBuffer) GetEntriesForTargets(targetKeys []stats.TargetKey) []LogEntry {
	entries := lb.GetEntries()
	var filtered []LogEntry

	targetKeyMap := make(map[string]bool)
	for _, key := range targetKeys {
		targetKeyMap[key.String()] = true
	}

	for _, entry := range entries {
		if targetKeyMap[entry.TargetKey.String()] {
			filtered = append(filtered, entry)
		}
	}

	return filtered
}

func (lb *LogBuffer) AddLogEntry(level LogLevel, message, details string, targetKey stats.TargetKey) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Details:   details,
		TargetKey: targetKey,
	}
	lb.Add(entry)
}

type nodeValue string

func (nv nodeValue) String() string {
	return string(nv)
}

func wrapText(text string, maxWidth int) []string {
	if len(text) <= maxWidth {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		for i := 0; i < len(text); i += maxWidth {
			end := min(i+maxWidth, len(text))
			lines = append(lines, text[i:end])
		}
		return lines
	}

	var currentLine strings.Builder

	for _, word := range words {
		switch {
		case currentLine.Len() == 0:
			currentLine.WriteString(word)
		case currentLine.Len()+1+len(word) <= maxWidth:
			currentLine.WriteString(" " + word)
		default:
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

func (m *Manager) updateLogsWidget(targetKey stats.TargetKey) {
	m.updateLogsWidgetForTargets([]stats.TargetKey{targetKey})
}

func (m *Manager) updateLogsWidgetForTargets(targetKeys []stats.TargetKey) {
	prevSelectedRow := m.detailsManager.LogsWidget.SelectedRow
	var prevSelectedValue string

	func() {
		defer func() {
			if r := recover(); r != nil {
				prevSelectedValue = ""
			}
		}()
		if prevSelectedNode := m.detailsManager.LogsWidget.SelectedNode(); prevSelectedNode != nil {
			prevSelectedValue = prevSelectedNode.Value.String()
		}
	}()

	expandedValues := make(map[string]bool)
	m.detailsManager.LogsWidget.Walk(func(node *widgets.TreeNode) bool {
		if node.Expanded {
			expandedValues[node.Value.String()] = true
		}
		return true
	})

	var logs []LogEntry
	if len(targetKeys) == 1 {
		logs = m.logBuffer.GetEntriesForTarget(targetKeys[0])
	} else {
		logs = m.logBuffer.GetEntriesForTargets(targetKeys)
	}

	if len(logs) == 0 {
		emptyNode := &widgets.TreeNode{
			Value: nodeValue("No logs available"),
			Nodes: []*widgets.TreeNode{},
		}
		m.detailsManager.LogsWidget.SetNodes([]*widgets.TreeNode{emptyNode})
		m.detailsManager.LogsWidget.Title = _recentLogsTitle
		return
	}

	maxLogs := _maxLogsToShow
	startIdx := 0
	if len(logs) > maxLogs {
		startIdx = len(logs) - maxLogs
	}

	var treeNodes []*widgets.TreeNode
	for _, log := range logs[startIdx:] {
		timeStr := log.Timestamp.Format("15:04:05")
		var levelPrefix string

		switch log.Level {
		case LogLevelError:
			levelPrefix = "[ERR]"
		case LogLevelWarning:
			levelPrefix = "[WRN]"
		case LogLevelInfo:
			levelPrefix = "[INF]"
		}

		mainText := fmt.Sprintf("%s [%s] %s", levelPrefix, timeStr, log.Message)

		var childNodes []*widgets.TreeNode
		if log.Details != "" {
			termWidth, _ := ui.TerminalDimensions()
			availableWidth := max(termWidth-50, 25)
			detailLines := wrapText(log.Details, availableWidth)

			detailsParent := &widgets.TreeNode{
				Value: nodeValue("Details:"),
				Nodes: []*widgets.TreeNode{},
			}

			for _, line := range detailLines {
				detailsParent.Nodes = append(detailsParent.Nodes, &widgets.TreeNode{
					Value: nodeValue(line),
					Nodes: []*widgets.TreeNode{},
				})
			}

			childNodes = append(childNodes, detailsParent)
		}

		childNodes = append(childNodes, &widgets.TreeNode{
			Value: nodeValue(fmt.Sprintf("Full timestamp: %s", log.Timestamp.Format("2006-01-02 15:04:05.000"))),
			Nodes: []*widgets.TreeNode{},
		})

		childNodes = append(childNodes, &widgets.TreeNode{
			Value: nodeValue(fmt.Sprintf("Target: %s", log.TargetKey.DisplayName())),
			Nodes: []*widgets.TreeNode{},
		})

		treeNode := &widgets.TreeNode{
			Value:    nodeValue(mainText),
			Nodes:    childNodes,
			Expanded: expandedValues[mainText],
		}
		treeNodes = append(treeNodes, treeNode)
	}

	for i, j := 0, len(treeNodes)-1; i < j; i, j = i+1, j-1 {
		treeNodes[i], treeNodes[j] = treeNodes[j], treeNodes[i]
	}

	m.detailsManager.LogsWidget.SetNodes(treeNodes)

	foundSelected := false
	for i, node := range treeNodes {
		if node.Value.String() == prevSelectedValue {
			m.detailsManager.LogsWidget.SelectedRow = i
			foundSelected = true
			break
		}
	}

	if !foundSelected && prevSelectedRow >= 0 && prevSelectedRow < len(treeNodes) {
		m.detailsManager.LogsWidget.SelectedRow = prevSelectedRow
	}

	title := fmt.Sprintf("Recent Logs (%d) - Press Enter to expand", len(logs))
	if len(targetKeys) > 1 {
		title = fmt.Sprintf("Recent Logs (%d from %d targets) - Press Enter to expand", len(logs), len(targetKeys))
	}
	m.detailsManager.LogsWidget.Title = title
}

func (m *Manager) NavigateLogs(direction int) {
	if !m.focusOnLogs || m.detailsManager.LogsWidget == nil {
		return
	}

	keys := m.getKeysForCurrentSelection()
	if len(keys) == 0 {
		return
	}

	var logs []LogEntry
	if len(keys) == 1 {
		logs = m.logBuffer.GetEntriesForTarget(keys[0])
	} else {
		logs = m.logBuffer.GetEntriesForTargets(keys)
	}

	if len(logs) == 0 {
		return
	}

	if direction > 0 {
		m.detailsManager.LogsWidget.ScrollDown()
	} else {
		m.detailsManager.LogsWidget.ScrollUp()
	}
}

func (m *Manager) IsFocusedOnLogs() bool {
	return m.focusOnLogs
}

func (m *Manager) IsLogsVisible() bool {
	return m.showLogs
}

func (m *Manager) ToggleLogsVisibility() {
	m.showLogs = !m.showLogs

	if m.showLogs {
		m.focusOnLogs = true
		m.detailsManager.ActiveGrid = m.detailsManager.LogsGrid

		m.detailsManager.LogsWidget.BorderStyle.Fg = ui.ColorGreen
		m.detailsManager.LogsWidget.Title = "Recent Logs (FOCUSED) - ↑↓:nav Enter:expand l:hide"

		keys := m.getKeysForCurrentSelection()
		if len(keys) > 0 {
			m.updateLogsWidgetForTargets(keys)
		} else {
			emptyNode := &widgets.TreeNode{
				Value: nodeValue("No target selected"),
				Nodes: []*widgets.TreeNode{},
			}
			m.detailsManager.LogsWidget.SetNodes([]*widgets.TreeNode{emptyNode})
			m.detailsManager.LogsWidget.Title = _recentLogsTitle
		}

		if m.listWidget != nil {
			m.listWidget.BorderStyle.Fg = ui.ColorCyan
		}
	} else {
		m.focusOnLogs = false
		m.detailsManager.ActiveGrid = m.detailsManager.NormalGrid

		if m.listWidget != nil {
			m.listWidget.BorderStyle.Fg = ui.ColorGreen
		}
	}

	m.setupGrid(m.termWidth, m.termHeight)
}
