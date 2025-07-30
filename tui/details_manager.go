package tui

import (
	"fmt"
	"time"

	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
	"github.com/Owloops/updo/utils"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

const (
	notAvailable = "N/A"
	checking     = "Checking..."
	passing      = "Passing"
	failing      = "Failing"
)

type DetailsPanelManager struct {
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
	TimingBreakdownWidget *TimingBreakdown
	Grid                  *ui.Grid
}

func NewDetailsPanelManager() *DetailsPanelManager {
	return &DetailsPanelManager{}
}

func (dm *DetailsPanelManager) Initialize(url string, refreshInterval time.Duration) {
	dm.initializeWidgets(url, refreshInterval)
	dm.setupGrid()
}

func (dm *DetailsPanelManager) initializeWidgets(url string, refreshInterval time.Duration) {
	dm.QuitWidget = widgets.NewParagraph()
	dm.QuitWidget.Title = "Information"
	dm.QuitWidget.Text = "Press q or <C-c> to quit"
	dm.QuitWidget.BorderStyle.Fg = ui.ColorClear

	dm.UptimeWidget = widgets.NewParagraph()
	dm.UptimeWidget.Title = "Uptime"
	dm.UptimeWidget.Text = "0%"
	dm.UptimeWidget.BorderStyle.Fg = ui.ColorCyan

	dm.UpForWidget = widgets.NewParagraph()
	dm.UpForWidget.Title = "Duration"
	dm.UpForWidget.Text = "0s"
	dm.UpForWidget.BorderStyle.Fg = ui.ColorBlue

	dm.AvgResponseTimeWidget = widgets.NewParagraph()
	dm.AvgResponseTimeWidget.Title = "Average"
	dm.AvgResponseTimeWidget.Text = notAvailable
	dm.AvgResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan

	dm.MinResponseTimeWidget = widgets.NewParagraph()
	dm.MinResponseTimeWidget.Title = "Min"
	dm.MinResponseTimeWidget.Text = notAvailable
	dm.MinResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan

	dm.MaxResponseTimeWidget = widgets.NewParagraph()
	dm.MaxResponseTimeWidget.Title = "Max"
	dm.MaxResponseTimeWidget.Text = notAvailable
	dm.MaxResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan

	dm.P95ResponseTimeWidget = widgets.NewParagraph()
	dm.P95ResponseTimeWidget.Title = "95p"
	dm.P95ResponseTimeWidget.Text = notAvailable
	dm.P95ResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan

	dm.SSLOkWidget = widgets.NewParagraph()
	dm.SSLOkWidget.Title = "SSL Certificate"
	dm.SSLOkWidget.Text = notAvailable
	dm.SSLOkWidget.BorderStyle.Fg = ui.ColorGreen

	dm.UptimePlot = widgets.NewPlot()
	dm.UptimePlot.Title = "Uptime History"
	dm.UptimePlot.Marker = widgets.MarkerDot
	dm.UptimePlot.BorderStyle.Fg = ui.ColorCyan
	dm.UptimePlot.Data = make([][]float64, 1)
	dm.UptimePlot.Data[0] = make([]float64, 0)
	dm.UptimePlot.LineColors[0] = ui.ColorCyan

	dm.ResponseTimePlot = widgets.NewPlot()
	dm.ResponseTimePlot.Title = "Response Time History"
	dm.ResponseTimePlot.Marker = widgets.MarkerBraille
	dm.ResponseTimePlot.BorderStyle.Fg = ui.ColorCyan
	dm.ResponseTimePlot.Data = make([][]float64, 1)
	dm.ResponseTimePlot.Data[0] = []float64{0.0, 0.0}
	dm.ResponseTimePlot.LineColors[0] = ui.ColorCyan

	dm.URLWidget = widgets.NewParagraph()
	dm.URLWidget.Title = "Monitoring Target"
	dm.URLWidget.Text = url
	dm.URLWidget.BorderStyle.Fg = ui.ColorBlue

	dm.RefreshWidget = widgets.NewParagraph()
	dm.RefreshWidget.Title = "Refresh Interval"
	dm.RefreshWidget.Text = fmt.Sprintf("%v seconds", refreshInterval.Seconds())
	dm.RefreshWidget.BorderStyle.Fg = ui.ColorBlue

	dm.AssertionWidget = widgets.NewParagraph()
	dm.AssertionWidget.Title = "Assertion Result"
	dm.AssertionWidget.Text = notAvailable
	dm.AssertionWidget.BorderStyle.Fg = ui.ColorCyan

	dm.TimingBreakdownWidget = NewTimingBreakdown()
	dm.TimingBreakdownWidget.Title = "Timing Breakdown"
	dm.TimingBreakdownWidget.BorderStyle.Fg = ui.ColorYellow
}

func (dm *DetailsPanelManager) setupGrid() {
	dm.Grid = ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	dm.Grid.SetRect(0, 0, termWidth, termHeight)

	dm.Grid.Set(
		ui.NewRow(1.0/7,
			ui.NewCol(1.0/4, dm.URLWidget),
			ui.NewCol(1.0/4, dm.RefreshWidget),
			ui.NewCol(1.0/4, dm.UpForWidget),
			ui.NewCol(1.0/4, dm.QuitWidget),
		),
		ui.NewRow(1.0/7,
			ui.NewCol(1.0/3, dm.UptimeWidget),
			ui.NewCol(1.0/3, dm.AssertionWidget),
			ui.NewCol(1.0/3, dm.SSLOkWidget),
		),
		ui.NewRow(5.0/7,
			ui.NewCol(3.0/5,
				ui.NewRow(0.5, dm.ResponseTimePlot),
				ui.NewRow(0.5, dm.UptimePlot),
			),
			ui.NewCol(2.0/5,
				ui.NewRow(0.5/2,
					ui.NewCol(1.0/2, dm.MinResponseTimeWidget),
					ui.NewCol(1.0/2, dm.MaxResponseTimeWidget),
				),
				ui.NewRow(0.5/2,
					ui.NewCol(1.0/2, dm.AvgResponseTimeWidget),
					ui.NewCol(1.0/2, dm.P95ResponseTimeWidget),
				),
				ui.NewRow(1.0/2, dm.TimingBreakdownWidget),
			),
		),
	)
}

func (dm *DetailsPanelManager) UpdateFromStats(stats stats.Stats) {
	dm.UptimeWidget.Text = fmt.Sprintf("%.2f%%", stats.UptimePercent)
	dm.UpForWidget.Text = utils.FormatDurationMinute(stats.TotalDuration)

	if stats.ChecksCount > 0 {
		dm.AvgResponseTimeWidget.Text = utils.FormatDurationMillisecond(stats.AvgResponseTime)
		dm.MinResponseTimeWidget.Text = utils.FormatDurationMillisecond(stats.MinResponseTime)
		dm.MaxResponseTimeWidget.Text = utils.FormatDurationMillisecond(stats.MaxResponseTime)
	} else {
		dm.AvgResponseTimeWidget.Text = notAvailable
		dm.MinResponseTimeWidget.Text = notAvailable
		dm.MaxResponseTimeWidget.Text = notAvailable
	}

	if stats.ChecksCount >= 2 {
		dm.P95ResponseTimeWidget.Text = fmt.Sprintf("%d ms", stats.P95.Milliseconds())
	} else {
		dm.P95ResponseTimeWidget.Text = notAvailable
	}
}

func (dm *DetailsPanelManager) UpdateFromResult(result net.WebsiteCheckResult, sslExpiry int) {
	if sslExpiry > 0 {
		dm.SSLOkWidget.Text = fmt.Sprintf("%d days remaining", sslExpiry)
	} else {
		dm.SSLOkWidget.Text = checking
	}

	switch {
	case result.AssertText == "":
		dm.AssertionWidget.Text = notAvailable
	case result.AssertionPassed:
		dm.AssertionWidget.Text = passing
	default:
		dm.AssertionWidget.Text = failing
	}

	if result.TraceInfo != nil {
		dm.TimingBreakdownWidget.SetTimings(map[string]time.Duration{
			"Wait":     result.TraceInfo.Wait,
			"DNS":      result.TraceInfo.DNSLookup,
			"TCP":      result.TraceInfo.TCPConnection,
			"TTFB":     result.TraceInfo.TimeToFirstByte,
			"Download": result.TraceInfo.DownloadDuration,
		})
	}
}

func (dm *DetailsPanelManager) UpdateTarget(url, region string, refreshInterval time.Duration) {
	if region != "" {
		dm.URLWidget.Title = "Target (Multi-Region)"
		dm.URLWidget.Text = fmt.Sprintf("%s\nActive Region: %s", url, region)
	} else {
		dm.URLWidget.Title = "Monitoring Target"
		dm.URLWidget.Text = url
	}

	dm.RefreshWidget.Text = fmt.Sprintf("%v seconds", refreshInterval.Seconds())
}

func (dm *DetailsPanelManager) UpdatePlots(result net.WebsiteCheckResult, width int) {
	dm.UptimePlot.Data[0] = append(dm.UptimePlot.Data[0], utils.BoolToFloat64(result.IsUp))
	dm.ResponseTimePlot.Data[0] = append(dm.ResponseTimePlot.Data[0], result.ResponseTime.Seconds())

	maxLength := width / 2

	if len(dm.UptimePlot.Data[0]) > maxLength {
		dm.UptimePlot.Data[0] = dm.UptimePlot.Data[0][len(dm.UptimePlot.Data[0])-maxLength:]
	}

	if len(dm.ResponseTimePlot.Data[0]) > maxLength {
		dm.ResponseTimePlot.Data[0] = dm.ResponseTimePlot.Data[0][len(dm.ResponseTimePlot.Data[0])-maxLength:]
	}
}

func (dm *DetailsPanelManager) ClearPlots() {
	dm.UptimePlot.Data[0] = make([]float64, 0)
	dm.ResponseTimePlot.Data[0] = []float64{0.0, 0.0}
}

func (dm *DetailsPanelManager) RestorePlots(uptimeData, responseTimeData []float64) {
	if uptimeData != nil {
		dm.UptimePlot.Data[0] = append([]float64{}, uptimeData...)
	} else {
		dm.UptimePlot.Data[0] = make([]float64, 0)
	}

	if responseTimeData != nil {
		dm.ResponseTimePlot.Data[0] = append([]float64{}, responseTimeData...)
	} else {
		dm.ResponseTimePlot.Data[0] = []float64{0.0, 0.0}
	}
}

func (dm *DetailsPanelManager) Resize(width, height int) {
	dm.Grid.SetRect(0, 0, width, height)
}
