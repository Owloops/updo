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
	Grid                  *ui.Grid
}

func NewDetailsManager() *DetailsManager {
	return &DetailsManager{}
}

func (m *DetailsManager) InitializeWidgets(url string, refreshInterval time.Duration) {
	m.QuitWidget = widgets.NewParagraph()
	m.QuitWidget.Title = "Information"
	m.QuitWidget.Text = "Press q or <C-c> to quit"
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

	m.Grid = ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	m.Grid.SetRect(0, 0, termWidth, termHeight)

	m.Grid.Set(
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
				ui.NewRow(0.5/2,
					ui.NewCol(1.0/2, m.MinResponseTimeWidget),
					ui.NewCol(1.0/2, m.MaxResponseTimeWidget),
				),
				ui.NewRow(0.5/2,
					ui.NewCol(1.0/2, m.AvgResponseTimeWidget),
					ui.NewCol(1.0/2, m.P95ResponseTimeWidget),
				),
				ui.NewRow(1.0/2, m.TimingBreakdownWidget),
			),
		),
	)
}

func (m *DetailsManager) UpdateWidgets(result net.WebsiteCheckResult, stats stats.Stats, width int, height int, manager *Manager) {

	m.UptimeWidget.Text = fmt.Sprintf("%.2f%%", stats.UptimePercent)
	m.UpForWidget.Text = utils.FormatDurationMinute(stats.TotalDuration)
	m.AvgResponseTimeWidget.Text = utils.FormatDurationMillisecond(stats.AvgResponseTime)
	m.MinResponseTimeWidget.Text = utils.FormatDurationMillisecond(stats.MinResponseTime)
	m.MaxResponseTimeWidget.Text = utils.FormatDurationMillisecond(stats.MaxResponseTime)

	if stats.ChecksCount >= 2 {
		m.P95ResponseTimeWidget.Text = fmt.Sprintf("%d ms", stats.P95.Milliseconds())
	}

	sslExpiry := manager.getSSLExpiry(result.URL)
	if sslExpiry > 0 {
		m.SSLOkWidget.Text = fmt.Sprintf("%d days remaining", sslExpiry)
	} else {
		m.SSLOkWidget.Text = checking
	}

	switch {
	case result.AssertText == "":
		m.AssertionWidget.Text = notAvailable
	case result.AssertionPassed:
		m.AssertionWidget.Text = passing
	default:
		m.AssertionWidget.Text = failing
	}

	if result.TraceInfo != nil {
		m.TimingBreakdownWidget.SetTimings(map[string]time.Duration{
			"Wait":     result.TraceInfo.Wait,
			"DNS":      result.TraceInfo.DNSLookup,
			"TCP":      result.TraceInfo.TCPConnection,
			"TTFB":     result.TraceInfo.TimeToFirstByte,
			"Download": result.TraceInfo.DownloadDuration,
		})
	}

	m.updatePlotsData(result, width)

	m.Grid.SetRect(0, 0, width, height)
	ui.Render(m.Grid)
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

func (m *DetailsManager) UpdateDurationWidgets(stats stats.Stats, width int, height int) {
	m.UpForWidget.Text = utils.FormatDurationMinute(stats.TotalDuration)

	m.Grid.SetRect(0, 0, width, height)
	ui.Render(m.Grid)
}

type Manager struct {
	targets        []config.Target
	targetData     map[string]TargetData
	plotData       map[string]PlotHistory
	sslExpiry      map[string]int
	sslExpiryMu    sync.RWMutex
	currentTarget  int
	isSingle       bool
	listWidget     *widgets.List
	detailsManager *DetailsManager
	grid           *ui.Grid
}

type PlotHistory struct {
	UptimeData       []float64
	ResponseTimeData []float64
}

func NewManager(targets []config.Target) *Manager {
	m := &Manager{
		targets:        targets,
		targetData:     make(map[string]TargetData),
		plotData:       make(map[string]PlotHistory),
		sslExpiry:      make(map[string]int),
		currentTarget:  0,
		isSingle:       len(targets) == 1,
		detailsManager: NewDetailsManager(),
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
	if len(m.targets) > 0 {
		m.detailsManager.InitializeWidgets(m.targets[0].URL, m.targets[0].GetRefreshInterval())
	}

	if !m.isSingle {
		m.listWidget = widgets.NewList()
		if len(m.targets) > 0 {
			m.listWidget.Title = fmt.Sprintf("Targets (↑↓) → %s", m.targets[0].Name)
		} else {
			m.listWidget.Title = "Targets"
		}
		m.listWidget.BorderStyle.Fg = ui.ColorCyan
		m.listWidget.TitleStyle.Fg = ui.ColorWhite
		m.listWidget.TitleStyle.Modifier = ui.ModifierBold

		m.updateTargetList()
	}

	m.setupGrid(width, height)
}

func (m *Manager) updateTargetList() {
	items := make([]string, len(m.targets))

	for i, target := range m.targets {
		icon := statusIcon
		statusColor := ""

		if data, exists := m.targetData[target.Name]; exists {
			if data.Result.IsUp {
				statusColor = " UP  "
			} else {
				statusColor = "DOWN "
			}
		} else {
			statusColor = "WAIT "
		}

		displayName := target.Name
		if displayName == "" || displayName == fmt.Sprintf("Target-%d", i+1) {
			displayName = target.URL
			if strings.HasPrefix(displayName, "https://") {
				displayName = displayName[8:]
			} else if strings.HasPrefix(displayName, "http://") {
				displayName = displayName[7:]
			}
		}

		if len(displayName) > 16 {
			displayName = displayName[:13] + "..."
		}

		if i == m.currentTarget {
			items[i] = fmt.Sprintf("▶ %s %s %s", icon, statusColor, displayName)
		} else {
			items[i] = fmt.Sprintf("  %s %s %s", icon, statusColor, displayName)
		}
	}

	m.listWidget.Rows = items
	m.listWidget.SelectedRow = m.currentTarget

	if m.currentTarget < len(m.targets) {
		if data, exists := m.targetData[m.targets[m.currentTarget].Name]; exists {
			if data.Result.IsUp {
				m.listWidget.SelectedRowStyle.Fg = ui.ColorGreen
			} else {
				m.listWidget.SelectedRowStyle.Fg = ui.ColorRed
			}
		} else {
			m.listWidget.SelectedRowStyle.Fg = ui.ColorYellow
		}
		m.listWidget.SelectedRowStyle.Modifier = ui.ModifierBold
	}
}

func (m *Manager) setupGrid(width, height int) {
	m.grid = ui.NewGrid()
	m.grid.SetRect(0, 0, width, height)

	if m.isSingle {
		m.grid.Set(
			ui.NewRow(1.0, m.detailsManager.Grid),
		)
	} else {
		m.grid.Set(
			ui.NewRow(1.0,
				ui.NewCol(0.22, m.listWidget),
				ui.NewCol(0.78, m.detailsManager.Grid),
			),
		)
	}

	ui.Render(m.grid)
}

func (m *Manager) SetActiveTarget(index int, monitors map[string]*stats.Monitor) {
	if index >= 0 && index < len(m.targets) {
		m.currentTarget = index
		target := m.targets[index]

		m.listWidget.Title = fmt.Sprintf("Targets (↑↓) → %s", target.Name)

		m.detailsManager.URLWidget.Text = target.URL
		m.detailsManager.RefreshWidget.Text = fmt.Sprintf("%v seconds", target.GetRefreshInterval().Seconds())

		m.restorePlotData(target.Name)
		m.updateTargetList()

		width, height := ui.TerminalDimensions()

		if monitor, exists := monitors[target.Name]; exists {
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
		}

		if data, exists := m.targetData[target.Name]; exists {
			sslExpiry := m.getSSLExpiry(data.Result.URL)
			if sslExpiry > 0 {
				m.detailsManager.SSLOkWidget.Text = fmt.Sprintf("%d days remaining", sslExpiry)
			} else {
				m.detailsManager.SSLOkWidget.Text = checking
			}

			switch {
			case data.Result.AssertText == "":
				m.detailsManager.AssertionWidget.Text = notAvailable
			case data.Result.AssertionPassed:
				m.detailsManager.AssertionWidget.Text = passing
			default:
				m.detailsManager.AssertionWidget.Text = failing
			}

			if data.Result.TraceInfo != nil {
				m.detailsManager.TimingBreakdownWidget.SetTimings(map[string]time.Duration{
					"Wait":     data.Result.TraceInfo.Wait,
					"DNS":      data.Result.TraceInfo.DNSLookup,
					"TCP":      data.Result.TraceInfo.TCPConnection,
					"TTFB":     data.Result.TraceInfo.TimeToFirstByte,
					"Download": data.Result.TraceInfo.DownloadDuration,
				})
			}
		}

		m.setupGrid(width, height)
	}
}

func (m *Manager) UpdateTarget(data TargetData) {
	m.targetData[data.Target.Name] = data

	m.updatePlotDataForTarget(data.Target.Name, data.Result)

	if m.targets[m.currentTarget].Name == data.Target.Name {
		m.restorePlotData(data.Target.Name)

		m.updateCurrentTargetWidgets(data.Result, data.Stats)

		if !m.isSingle {
			m.updateTargetList()
		}
		ui.Render(m.grid)
	} else if !m.isSingle {
		m.updateTargetList()
	}
}

func (m *Manager) RefreshStats(monitors map[string]*stats.Monitor) {
	currentTargetName := m.targets[m.currentTarget].Name
	if monitor, exists := monitors[currentTargetName]; exists {
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

	width, _ := ui.TerminalDimensions()
	m.detailsManager.updatePlotsData(result, width)
}

func (m *Manager) Resize(width, height int) {
	m.setupGrid(width, height)
}

func (m *Manager) restorePlotData(targetName string) {
	if history, exists := m.plotData[targetName]; exists {
		m.detailsManager.UptimePlot.Data[0] = append([]float64{}, history.UptimeData...)
		m.detailsManager.ResponseTimePlot.Data[0] = append([]float64{}, history.ResponseTimeData...)
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

	width, _ := ui.TerminalDimensions()
	maxLength := width / 2

	if len(history.UptimeData) > maxLength {
		history.UptimeData = history.UptimeData[len(history.UptimeData)-maxLength:]
	}

	if len(history.ResponseTimeData) > maxLength {
		history.ResponseTimeData = history.ResponseTimeData[len(history.ResponseTimeData)-maxLength:]
	}

	m.plotData[targetName] = history
}
