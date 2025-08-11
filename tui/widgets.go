package tui

import (
	"fmt"
	"time"

	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/utils"
	uw "github.com/Owloops/updo/widgets"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

const _recentLogsTitle = "Recent Logs"

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
	m.AvgResponseTimeWidget.Text = _notAvailable
	m.AvgResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan

	m.MinResponseTimeWidget = widgets.NewParagraph()
	m.MinResponseTimeWidget.Title = "Min"
	m.MinResponseTimeWidget.Text = _notAvailable
	m.MinResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan

	m.MaxResponseTimeWidget = widgets.NewParagraph()
	m.MaxResponseTimeWidget.Title = "Max"
	m.MaxResponseTimeWidget.Text = _notAvailable
	m.MaxResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan

	m.P95ResponseTimeWidget = widgets.NewParagraph()
	m.P95ResponseTimeWidget.Title = "95p"
	m.P95ResponseTimeWidget.Text = _notAvailable
	m.P95ResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan

	m.SSLOkWidget = widgets.NewParagraph()
	m.SSLOkWidget.Title = "SSL Certificate"
	m.SSLOkWidget.Text = _notAvailable
	m.SSLOkWidget.BorderStyle.Fg = ui.ColorGreen

	m.UptimePlot = widgets.NewPlot()
	m.UptimePlot.Title = "Uptime History"
	m.UptimePlot.Marker = widgets.MarkerDot
	m.UptimePlot.BorderStyle.Fg = ui.ColorCyan
	m.UptimePlot.Data = make([][]float64, 1)
	m.UptimePlot.Data[0] = nil
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
	m.AssertionWidget.Text = _notAvailable
	m.AssertionWidget.BorderStyle.Fg = ui.ColorCyan

	m.TimingBreakdownWidget = uw.NewTimingBreakdown()
	m.TimingBreakdownWidget.Title = "Timing Breakdown"
	m.TimingBreakdownWidget.BorderStyle.Fg = ui.ColorYellow

	m.LogsWidget = widgets.NewTree()
	m.LogsWidget.Title = _recentLogsTitle
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
