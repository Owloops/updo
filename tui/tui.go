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

type DetailsManager struct {
	QuitWidget            *widgets.Paragraph
	UptimeWidget          *widgets.Paragraph
	UpForWidget           *widgets.Paragraph
	AvgResponseTimeWidget *widgets.Paragraph
	MinResponseTimeWidget *widgets.Paragraph
	MaxResponseTimeWidget *widgets.Paragraph
	P95ResponseTimeWidget *widgets.Paragraph
	SSLOkWidget           *widgets.Paragraph
	UptimePlot            *widgets.Plot
	ResponseTimePlot      *widgets.Plot
	URLWidget             *widgets.Paragraph
	RefreshWidget         *widgets.Paragraph
	AssertionWidget       *widgets.Paragraph
	TimingBreakdownWidget *uw.TimingBreakdown
	LogsWidget            *widgets.Tree
	NormalGrid            *ui.Grid
	LogsGrid              *ui.Grid
	ActiveGrid            *ui.Grid
}

func NewDetailsManager() *DetailsManager {
	return &DetailsManager{}
}

func (m *DetailsManager) InitializeWidgets(url string, refreshInterval time.Duration) {
	m.QuitWidget = widgets.NewParagraph()
	m.QuitWidget.Title = "Information"
	m.QuitWidget.Text = "q:quit l:logs ↑↓:nav"
	m.QuitWidget.BorderStyle.Fg = ui.ColorClear

	m.UptimeWidget = widgets.NewParagraph()
	m.UptimeWidget.Title = "Uptime"
	m.UptimeWidget.Text = "0%"
	m.UptimeWidget.BorderStyle.Fg = ui.ColorCyan

	m.UpForWidget = widgets.NewParagraph()
	m.UpForWidget.Title = "Duration"
	m.UpForWidget.Text = "0s"
	m.UpForWidget.BorderStyle.Fg = ui.ColorBlue

	m.AvgResponseTimeWidget = widgets.NewParagraph()
	m.AvgResponseTimeWidget.Title = "Average"
	m.AvgResponseTimeWidget.Text = notAvailable
	m.AvgResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan

	m.MinResponseTimeWidget = widgets.NewParagraph()
	m.MinResponseTimeWidget.Title = "Min"
	m.MinResponseTimeWidget.Text = notAvailable
	m.MinResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan

	m.MaxResponseTimeWidget = widgets.NewParagraph()
	m.MaxResponseTimeWidget.Title = "Max"
	m.MaxResponseTimeWidget.Text = notAvailable
	m.MaxResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan

	m.P95ResponseTimeWidget = widgets.NewParagraph()
	m.P95ResponseTimeWidget.Title = "95p"
	m.P95ResponseTimeWidget.Text = notAvailable
	m.P95ResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan

	m.SSLOkWidget = widgets.NewParagraph()
	m.SSLOkWidget.Title = "SSL Certificate"
	m.SSLOkWidget.Text = notAvailable
	m.SSLOkWidget.BorderStyle.Fg = ui.ColorGreen

	m.UptimePlot = widgets.NewPlot()
	m.UptimePlot.Title = "Uptime History"
	m.UptimePlot.Marker = widgets.MarkerDot
	m.UptimePlot.BorderStyle.Fg = ui.ColorCyan
	m.UptimePlot.Data = make([][]float64, 1)
	m.UptimePlot.Data[0] = make([]float64, 0)
	m.UptimePlot.LineColors[0] = ui.ColorCyan

	m.ResponseTimePlot = widgets.NewPlot()
	m.ResponseTimePlot.Title = "Response Time History"
	m.ResponseTimePlot.Marker = widgets.MarkerBraille
	m.ResponseTimePlot.BorderStyle.Fg = ui.ColorCyan
	m.ResponseTimePlot.Data = make([][]float64, 1)
	m.ResponseTimePlot.Data[0] = []float64{0.0, 0.0}
	m.ResponseTimePlot.LineColors[0] = ui.ColorCyan

	m.URLWidget = widgets.NewParagraph()
	m.URLWidget.Title = "Monitoring URL"
	m.URLWidget.Text = url
	m.URLWidget.BorderStyle.Fg = ui.ColorBlue

	m.RefreshWidget = widgets.NewParagraph()
	m.RefreshWidget.Title = "Refresh Interval"
	m.RefreshWidget.Text = fmt.Sprintf("%v seconds", refreshInterval.Seconds())
	m.RefreshWidget.BorderStyle.Fg = ui.ColorBlue

	m.AssertionWidget = widgets.NewParagraph()
	m.AssertionWidget.Title = "Assertion Result"
	m.AssertionWidget.Text = notAvailable
	m.AssertionWidget.BorderStyle.Fg = ui.ColorCyan

	m.TimingBreakdownWidget = uw.NewTimingBreakdown()
	m.TimingBreakdownWidget.Title = "Timing Breakdown"
	m.TimingBreakdownWidget.BorderStyle.Fg = ui.ColorYellow

	m.LogsWidget = widgets.NewTree()
	m.LogsWidget.Title = "Recent Logs"
	m.LogsWidget.BorderStyle.Fg = ui.ColorMagenta
	m.LogsWidget.TitleStyle.Fg = ui.ColorWhite
	m.LogsWidget.TitleStyle.Modifier = ui.ModifierBold
	m.LogsWidget.TextStyle.Fg = ui.ColorWhite
	m.LogsWidget.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorMagenta, ui.ModifierBold)

	termWidth, termHeight := ui.TerminalDimensions()

	m.NormalGrid = ui.NewGrid()
	m.NormalGrid.SetRect(0, 0, termWidth, termHeight)
	m.setupNormalGrid()

	m.LogsGrid = ui.NewGrid()
	m.LogsGrid.SetRect(0, 0, termWidth, termHeight)
	m.setupLogsGrid()

	m.ActiveGrid = m.NormalGrid
}

func (m *DetailsManager) setupNormalGrid() {
	m.NormalGrid.Set(
		ui.NewRow(1.0/7,
			ui.NewCol(1.0/4, m.URLWidget),
			ui.NewCol(1.0/4, m.RefreshWidget),
			ui.NewCol(1.0/4, m.UpForWidget),
			ui.NewCol(1.0/4, m.QuitWidget),
		),
		ui.NewRow(1.0/7,
			ui.NewCol(1.0/3, m.UptimeWidget),
			ui.NewCol(1.0/3, m.AssertionWidget),
			ui.NewCol(1.0/3, m.SSLOkWidget),
		),
		ui.NewRow(5.0/7,
			ui.NewCol(3.0/5,
				ui.NewRow(0.5, m.ResponseTimePlot),
				ui.NewRow(0.5, m.UptimePlot),
			),
			ui.NewCol(2.0/5,
				ui.NewRow(0.5,
					ui.NewCol(1.0/2,
						ui.NewRow(0.5, m.MinResponseTimeWidget),
						ui.NewRow(0.5, m.AvgResponseTimeWidget),
					),
					ui.NewCol(1.0/2,
						ui.NewRow(0.5, m.MaxResponseTimeWidget),
						ui.NewRow(0.5, m.P95ResponseTimeWidget),
					),
				),
				ui.NewRow(0.5, m.TimingBreakdownWidget),
			),
		),
	)
}

func (m *DetailsManager) setupLogsGrid() {
	m.LogsGrid.Set(
		ui.NewRow(0.1,
			ui.NewCol(1.0/4, m.URLWidget),
			ui.NewCol(1.0/4, m.RefreshWidget),
			ui.NewCol(1.0/4, m.UpForWidget),
			ui.NewCol(1.0/4, m.QuitWidget),
		),
		ui.NewRow(0.1,
			ui.NewCol(1.0/3, m.UptimeWidget),
			ui.NewCol(1.0/3, m.AssertionWidget),
			ui.NewCol(1.0/3, m.SSLOkWidget),
		),
		ui.NewRow(0.5,
			ui.NewCol(3.0/5,
				ui.NewRow(0.5, m.ResponseTimePlot),
				ui.NewRow(0.5, m.UptimePlot),
			),
			ui.NewCol(2.0/5,
				ui.NewRow(0.5,
					ui.NewCol(1.0/2,
						ui.NewRow(0.5, m.MinResponseTimeWidget),
						ui.NewRow(0.5, m.AvgResponseTimeWidget),
					),
					ui.NewCol(1.0/2,
						ui.NewRow(0.5, m.MaxResponseTimeWidget),
						ui.NewRow(0.5, m.P95ResponseTimeWidget),
					),
				),
				ui.NewRow(0.5, m.TimingBreakdownWidget),
			),
		),
		ui.NewRow(0.3, m.LogsWidget),
	)
}

func (m *DetailsManager) updatePlotsData(result net.WebsiteCheckResult, width int) {
	m.UptimePlot.Data[0] = append(m.UptimePlot.Data[0], utils.BoolToFloat64(result.IsUp))
	m.ResponseTimePlot.Data[0] = append(m.ResponseTimePlot.Data[0], result.ResponseTime.Seconds())

	maxLength := width / 2

	if len(m.UptimePlot.Data[0]) > maxLength {
		m.UptimePlot.Data[0] = m.UptimePlot.Data[0][len(m.UptimePlot.Data[0])-maxLength:]
	}

	if len(m.ResponseTimePlot.Data[0]) > maxLength {
		m.ResponseTimePlot.Data[0] = m.ResponseTimePlot.Data[0][len(m.ResponseTimePlot.Data[0])-maxLength:]
	}
}

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
		m.searchWidget = widgets.NewParagraph()
		m.searchWidget.Border = true
		m.searchWidget.BorderStyle.Fg = ui.ColorCyan
		m.searchWidget.Title = "Search"
		m.searchWidget.TitleStyle.Fg = ui.ColorWhite
		m.searchWidget.TitleStyle.Modifier = ui.ModifierBold
		m.searchWidget.Text = "Press / to activate"
		m.searchWidget.TextStyle.Fg = ui.ColorWhite

		m.listWidget = uw.NewFilteredList()
		if len(allKeys) > 0 {
			m.listWidget.Title = fmt.Sprintf("Targets → %s", allKeys[0].DisplayName())
		} else {
			m.listWidget.Title = "Targets"
		}
		m.listWidget.BorderStyle.Fg = ui.ColorCyan
		m.listWidget.TitleStyle.Fg = ui.ColorWhite
		m.listWidget.TitleStyle.Modifier = ui.ModifierBold

		m.listWidget.OnSearchChange = func(query string, filteredIndices []int) {
			if m.listWidget.IsSearchMode() {
				if query != "" {
					m.searchWidget.Text = query
					m.searchWidget.TextStyle.Fg = ui.ColorGreen
					m.searchWidget.BorderStyle.Fg = ui.ColorGreen
					m.listWidget.Title = fmt.Sprintf("Targets (%d/%d)", len(filteredIndices), len(allKeys))
				} else {
					m.searchWidget.Text = "Type to filter..."
					m.searchWidget.TextStyle.Fg = ui.ColorYellow
					m.searchWidget.BorderStyle.Fg = ui.ColorYellow
					m.listWidget.Title = "Targets"
				}

				keyVisible := false
				if len(filteredIndices) > 0 {
					for i, idx := range filteredIndices {
						if idx == m.currentKeyIndex {
							m.listWidget.SelectedRow = i
							keyVisible = true
							break
						}
					}
				}

				if !keyVisible {
					m.listWidget.SelectedRow = 0
					m.listWidget.SelectedRowStyle = m.listWidget.TextStyle
				}
			} else {
				m.searchWidget.Text = "Press / to activate"
				m.searchWidget.TextStyle.Fg = ui.ColorWhite
				m.searchWidget.BorderStyle.Fg = ui.ColorCyan
				if m.currentKeyIndex < len(allKeys) {
					m.listWidget.Title = fmt.Sprintf("Targets → %s", allKeys[m.currentKeyIndex].DisplayName())
				}
				m.listWidget.SelectedRow = m.currentKeyIndex
			}
		}

		m.updateTargetList()
	}

	m.setupGrid(width, height)
}

func (m *Manager) updateTargetList() {
	allKeys := m.keyRegistry.GetAllKeys()
	items := make([]string, len(allKeys))

	for i, key := range allKeys {
		icon := statusIcon
		statusColor := ""

		if data, exists := m.targetData[key.String()]; exists {
			if data.Result.IsUp {
				statusColor = " UP  "
			} else {
				statusColor = "DOWN "
			}
		} else {
			statusColor = "WAIT "
		}

		displayName := key.DisplayName()
		maxLen := 25
		if len(displayName) > maxLen {
			displayName = displayName[:maxLen-3] + "..."
		}

		if i == m.currentKeyIndex {
			items[i] = fmt.Sprintf("▶ %s %s %s", icon, statusColor, displayName)
		} else {
			items[i] = fmt.Sprintf("  %s %s %s", icon, statusColor, displayName)
		}
	}

	m.listWidget.SetRows(items)

	keyVisible := true
	if m.listWidget.IsSearchMode() {
		indices := m.listWidget.GetFilteredIndices()
		keyVisible = slices.Contains(indices, m.currentKeyIndex)
	}

	if keyVisible && m.currentKeyIndex < len(allKeys) {
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
		m.listWidget.SelectedRowStyle = m.listWidget.TextStyle
	}
}

func (m *Manager) setupGrid(width, height int) {
	m.grid = ui.NewGrid()
	m.grid.SetRect(0, 0, width, height)

	if m.isSingle {
		m.grid.Set(
			ui.NewRow(1.0, m.detailsManager.ActiveGrid),
		)
	} else {
		m.grid.Set(
			ui.NewRow(1.0,
				ui.NewCol(0.22,
					ui.NewRow(1.0/7, m.searchWidget),
					ui.NewRow(6.0/7, m.listWidget),
				),
				ui.NewCol(0.78, m.detailsManager.ActiveGrid),
			),
		)
	}

	ui.Render(m.grid)
}

func (m *Manager) SetActiveTargetKey(keyIndex int, monitors map[string]*stats.Monitor) {
	allKeys := m.keyRegistry.GetAllKeys()
	if keyIndex >= 0 && keyIndex < len(allKeys) {
		m.currentKeyIndex = keyIndex
		m.updateActiveTarget(monitors)
	}
}

func (m *Manager) UpdateTarget(data TargetData) {
	targetKeyStr := data.TargetKey.String()
	m.targetData[targetKeyStr] = data

	m.updatePlotDataForTarget(targetKeyStr, data.Result)

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
	if m.currentKeyIndex < len(allKeys) {
		currentKey := allKeys[m.currentKeyIndex]
		if currentKey.String() == targetKeyStr {
			m.restorePlotData(targetKeyStr)
			m.updateCurrentTargetWidgets(data.Result, data.Stats)
			if m.showLogs {
				m.updateLogsWidget(currentKey)
			}
			if !m.isSingle {
				m.updateTargetList()
			}
			ui.Render(m.grid)
		} else if !m.isSingle {
			m.updateTargetList()
		}
	}
}

type nodeValue string

func (nv nodeValue) String() string {
	return string(nv)
}

func (m *Manager) updateLogsWidget(targetKey TargetKey) {
	logs := m.logBuffer.GetEntriesForTarget(targetKey)

	if len(logs) == 0 {
		emptyNode := &widgets.TreeNode{
			Value: nodeValue("No logs available"),
			Nodes: []*widgets.TreeNode{},
		}
		m.detailsManager.LogsWidget.SetNodes([]*widgets.TreeNode{emptyNode})
		m.detailsManager.LogsWidget.Title = "Recent Logs"
		return
	}

	maxLogs := 10
	startIdx := 0
	if len(logs) > maxLogs {
		startIdx = len(logs) - maxLogs
	}

	var treeNodes []*widgets.TreeNode
	for _, log := range logs[startIdx:] {
		timeStr := log.Timestamp.Format("15:04:05")
		levelColor := ""
		levelStr := string(log.Level)

		switch log.Level {
		case LogLevelError:
			levelColor = "[red]"
			levelStr = "ERROR"
		case LogLevelWarning:
			levelColor = "[yellow]"
			levelStr = "WARN"
		case LogLevelInfo:
			levelColor = "[green]"
			levelStr = "INFO"
		}

		mainText := fmt.Sprintf("%s%s[white] [%s] %s", levelColor, levelStr, timeStr, log.Message)

		var childNodes []*widgets.TreeNode
		if log.Details != "" {
			childNodes = append(childNodes, &widgets.TreeNode{
				Value: nodeValue(fmt.Sprintf("Details: %s", log.Details)),
				Nodes: []*widgets.TreeNode{},
			})
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
			Value: nodeValue(mainText),
			Nodes: childNodes,
		}
		treeNodes = append(treeNodes, treeNode)
	}

	for i, j := 0, len(treeNodes)-1; i < j; i, j = i+1, j-1 {
		treeNodes[i], treeNodes[j] = treeNodes[j], treeNodes[i]
	}

	m.detailsManager.LogsWidget.SetNodes(treeNodes)
	m.detailsManager.LogsWidget.Title = fmt.Sprintf("Recent Logs (%d) - Press Enter to expand", len(logs))
}

func (m *Manager) NavigateLogs(direction int) {
	if !m.focusOnLogs || m.detailsManager.LogsWidget == nil {
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

		allKeys := m.keyRegistry.GetAllKeys()
		if m.currentKeyIndex < len(allKeys) {
			currentKey := allKeys[m.currentKeyIndex]
			m.updateLogsWidget(currentKey)
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

func (m *Manager) RefreshStats(monitors map[string]*stats.Monitor) {
	allKeys := m.keyRegistry.GetAllKeys()
	if m.currentKeyIndex >= len(allKeys) {
		return
	}

	currentKey := allKeys[m.currentKeyIndex]
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
			m.detailsManager.P95ResponseTimeWidget.Text = notAvailable
		}

		if !m.isSingle {
			m.updateTargetList()
		}
		ui.Render(m.grid)
	}
}

func (m *Manager) updateCurrentTargetWidgets(result net.WebsiteCheckResult, stats stats.Stats) {
	m.detailsManager.UptimeWidget.Text = fmt.Sprintf("%.2f%%", stats.UptimePercent)
	m.detailsManager.UpForWidget.Text = utils.FormatDurationMinute(stats.TotalDuration)
	m.detailsManager.AvgResponseTimeWidget.Text = utils.FormatDurationMillisecond(stats.AvgResponseTime)
	m.detailsManager.MinResponseTimeWidget.Text = utils.FormatDurationMillisecond(stats.MinResponseTime)
	m.detailsManager.MaxResponseTimeWidget.Text = utils.FormatDurationMillisecond(stats.MaxResponseTime)

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

func (m *Manager) Resize(width, height int) {
	m.termWidth = width
	m.termHeight = height
	m.grid.SetRect(0, 0, width, height)
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
