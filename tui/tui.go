package tui

import (
	"fmt"
	"math"
	"time"

	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/utils"
	uw "github.com/Owloops/updo/widgets"

	"github.com/caio/go-tdigest/v4"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type Manager struct {
	ChecksCount       int
	TotalResponseTime time.Duration
	TotalUptime       time.Duration
	StartTime         time.Time
	LastCheckTime     time.Time
	IsUp              bool
	MinResponseTime   float64
	MaxResponseTime   float64
	TDigest           *tdigest.TDigest

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

func NewManager() *Manager {
	td, err := tdigest.New(tdigest.Compression(100))
	if err != nil {
	}
	return &Manager{
		StartTime:       time.Now(),
		MinResponseTime: math.MaxFloat64,
		MaxResponseTime: 0,
		TDigest:         td,
	}
}

func (m *Manager) InitializeWidgets(url string, refreshInterval time.Duration) {
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
	m.AvgResponseTimeWidget.Text = "N/A"
	m.AvgResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan

	m.MinResponseTimeWidget = widgets.NewParagraph()
	m.MinResponseTimeWidget.Title = "Min"
	m.MinResponseTimeWidget.Text = "N/A"
	m.MinResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan

	m.MaxResponseTimeWidget = widgets.NewParagraph()
	m.MaxResponseTimeWidget.Title = "Max"
	m.MaxResponseTimeWidget.Text = "N/A"
	m.MaxResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan

	m.P95ResponseTimeWidget = widgets.NewParagraph()
	m.P95ResponseTimeWidget.Title = "95p"
	m.P95ResponseTimeWidget.Text = "N/A"
	m.P95ResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan

	m.SSLOkWidget = widgets.NewParagraph()
	m.SSLOkWidget.Title = "SSL Certificate"
	m.SSLOkWidget.Text = "N/A"
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
	m.AssertionWidget.Text = "N/A"
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

func (m *Manager) UpdateWidgets(result net.WebsiteCheckResult, width int, height int) {
	m.ChecksCount++
	uptimePercentage := m.calculateUptimePercentage(result.IsUp)

	m.UptimeWidget.Text = fmt.Sprintf("%.2f%%", uptimePercentage)

	totalMonitoringTime := time.Since(m.StartTime)
	m.UpForWidget.Text = utils.FormatDurationMinute(totalMonitoringTime)

	m.TotalResponseTime += result.ResponseTime
	avgResponseTime := m.TotalResponseTime / time.Duration(m.ChecksCount)
	m.AvgResponseTimeWidget.Text = utils.FormatDurationMillisecond(avgResponseTime)

	if m.ChecksCount == 1 || result.ResponseTime < time.Duration(m.MinResponseTime) {
		m.MinResponseTime = float64(result.ResponseTime)
	}
	if result.ResponseTime > time.Duration(m.MaxResponseTime) {
		m.MaxResponseTime = float64(result.ResponseTime)
	}

	m.MinResponseTimeWidget.Text = utils.FormatDurationMillisecond(time.Duration(m.MinResponseTime))
	m.MaxResponseTimeWidget.Text = utils.FormatDurationMillisecond(time.Duration(m.MaxResponseTime))

	err := m.TDigest.Add(result.ResponseTime.Seconds())
	if err != nil {
	}

	p95 := int(m.TDigest.Quantile(0.95) * 1000)
	m.P95ResponseTimeWidget.Text = fmt.Sprintf("%d ms", p95)

	sslExpiry := net.GetSSLCertExpiry(result.URL)
	m.SSLOkWidget.Text = fmt.Sprintf("%d days remaining", sslExpiry)

	if result.AssertText == "" {
		m.AssertionWidget.Text = "N/A"
	} else if result.AssertionPassed {
		m.AssertionWidget.Text = "Passing"
	} else {
		m.AssertionWidget.Text = "Failing"
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

func (m *Manager) updatePlotsData(result net.WebsiteCheckResult, width int) {
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

func (m *Manager) UpdateDurationWidgets(width int, height int) {
	totalMonitoringTime := time.Since(m.StartTime)
	m.UpForWidget.Text = utils.FormatDurationMinute(totalMonitoringTime)

	m.Grid.SetRect(0, 0, width, height)
	ui.Render(m.Grid)
}

func (m *Manager) calculateUptimePercentage(isUp bool) float64 {
	now := time.Now()
	totalMonitoredTime := now.Sub(m.StartTime)
	if m.ChecksCount == 1 {
		m.LastCheckTime = now
		if isUp {
			m.TotalUptime = now.Sub(m.StartTime)
		}
	} else {
		timeElapsedSinceLastCheck := now.Sub(m.LastCheckTime)
		m.LastCheckTime = now

		if isUp {
			m.TotalUptime += timeElapsedSinceLastCheck
		}
	}
	m.IsUp = isUp
	if totalMonitoredTime == 0 {
		return 0
	}
	return (float64(m.TotalUptime) / float64(totalMonitoredTime)) * 100
}
