package tui

import (
	"fmt"
	"strconv"
	"time"

	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/utils"
	uw "github.com/Owloops/updo/widgets"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

var (
	checksCount       int
	totalResponseTime time.Duration
	totalUptime       time.Duration

	quitWidget            *widgets.Paragraph
	uptimeWidget          *widgets.Paragraph
	upForWidget           *widgets.Paragraph
	avgResponseTimeWidget *widgets.Paragraph
	sslExpiryWidget       *widgets.Paragraph
	uptimePlot            *widgets.Plot
	responseTimePlot      *widgets.Plot
	urlWidget             *widgets.Paragraph
	refreshWidget         *widgets.Paragraph
	assertionWidget       *widgets.Paragraph
	timingBreakdownWidget *uw.TimingBreakdown
	grid                  *ui.Grid
)

func InitializeWidgets() {

	quitWidget := widgets.NewParagraph()
	quitWidget.Title = "Information"
	quitWidget.Text = "Press q or <C-c> to quit"
	quitWidget.BorderStyle.Fg = ui.ColorCyan

	uptimeWidget = widgets.NewParagraph()
	uptimeWidget.Title = "Uptime"
	uptimeWidget.BorderStyle.Fg = ui.ColorCyan
	uptimeWidget.Text = ""

	upForWidget = widgets.NewParagraph()
	upForWidget.Title = "Duration"
	upForWidget.BorderStyle.Fg = ui.ColorCyan
	upForWidget.Text = ""

	avgResponseTimeWidget = widgets.NewParagraph()
	avgResponseTimeWidget.Title = "Average Response Time"
	avgResponseTimeWidget.BorderStyle.Fg = ui.ColorCyan
	avgResponseTimeWidget.Text = ""

	sslExpiryWidget = widgets.NewParagraph()
	sslExpiryWidget.Title = "SSL days remaining"
	sslExpiryWidget.BorderStyle.Fg = ui.ColorGreen
	sslExpiryWidget.Text = ""

	uptimePlot = widgets.NewPlot()
	uptimePlot.Title = "Uptime History"
	uptimePlot.Marker = widgets.MarkerDot
	uptimePlot.BorderStyle.Fg = ui.ColorCyan
	uptimePlot.Data = make([][]float64, 1)
	uptimePlot.Data[0] = make([]float64, 0)
	uptimePlot.LineColors[0] = ui.ColorCyan

	responseTimePlot = widgets.NewPlot()
	responseTimePlot.Title = "Response Time History"
	responseTimePlot.Marker = widgets.MarkerBraille
	responseTimePlot.BorderStyle.Fg = ui.ColorCyan
	responseTimePlot.Data = make([][]float64, 1)
	responseTimePlot.Data[0] = []float64{0.0}
	responseTimePlot.LineColors[0] = ui.ColorCyan

	urlWidget = widgets.NewParagraph()
	urlWidget.Title = "Monitoring URL"
	urlWidget.BorderStyle.Fg = ui.ColorBlue

	refreshWidget = widgets.NewParagraph()
	refreshWidget.Title = "Refresh Interval"
	refreshWidget.BorderStyle.Fg = ui.ColorBlue

	assertionWidget = widgets.NewParagraph()
	assertionWidget.Title = "Assertion Result"
	assertionWidget.BorderStyle.Fg = ui.ColorBlue

	timingBreakdownWidget = uw.NewTimingBreakdown()
	timingBreakdownWidget.Title = "Timing Breakdown"
	timingBreakdownWidget.BorderStyle.Fg = ui.ColorYellow

	grid = ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	grid.Set(
		ui.NewRow(1.0/7,
			ui.NewCol(1.0/4, urlWidget),
			ui.NewCol(1.0/4, refreshWidget),
			ui.NewCol(1.0/4, assertionWidget),
			ui.NewCol(1.0/4, quitWidget),
		),
		ui.NewRow(1.0/7,
			ui.NewCol(1.0/4, uptimeWidget),
			ui.NewCol(1.0/4, upForWidget),
			ui.NewCol(1.0/4, avgResponseTimeWidget),
			ui.NewCol(1.0/4, sslExpiryWidget),
		),
		ui.NewRow(5.0/7,
			ui.NewCol(3.0/5,
				ui.NewRow(0.5, responseTimePlot),
				ui.NewRow(0.5, uptimePlot),
			),
			ui.NewCol(2.0/5,
				ui.NewRow(1.0, timingBreakdownWidget),
			),
		),
	)

}

func PerformCheckAndUpdateWidgets(url string, shouldFail bool, timeout time.Duration, followRedirects bool, skipSSL bool, assertText string, refreshFlag int, lastCheckTime time.Time, startTime time.Time, width int, height int) (time.Time, bool) {
	isUp, responseTime, traceInfo, assertionPassed := net.CheckWebsite(url, shouldFail, timeout, followRedirects, skipSSL, assertText)
	sslExpiry := net.GetSSLCertExpiry(url)
	checksCount++
	totalResponseTime += responseTime

	now := time.Now()
	timeSinceLastCheck := now.Sub(lastCheckTime)
	lastCheckTime = now

	if isUp {
		totalUptime += timeSinceLastCheck
	}

	totalMonitoringTime := time.Since(startTime)

	uptimePercentage := 0.0
	if totalMonitoringTime > 0 {
		uptimePercentage = (float64(totalUptime) / float64(totalMonitoringTime)) * 100
	}

	urlWidget.Text = url
	refreshWidget.Text = strconv.Itoa(refreshFlag) + "s"
	uptimeWidget.Text = fmt.Sprintf("%.2f%%", uptimePercentage)
	upForWidget.Text = utils.FormatDurationMinute(totalMonitoringTime)

	if checksCount > 0 {
		avgResponseTimeWidget.Text = utils.FormatDurationMillisecond(totalResponseTime / time.Duration(checksCount))
	} else {
		avgResponseTimeWidget.Text = "N/A"
	}

	sslExpiryWidget.Text = utils.FormatInt(sslExpiry)

	if traceInfo != nil {
		timingsMap := map[string]time.Duration{
			"Wait":     traceInfo.Wait,
			"DNS":      traceInfo.DNSLookup,
			"TCP":      traceInfo.TCPConnection,
			"TTFB":     traceInfo.TimeToFirstByte,
			"Download": traceInfo.DownloadDuration,
		}
		timingBreakdownWidget.SetTimings(timingsMap)
	}

	if assertText == "" {
		assertionWidget.Text = "N/A"
	} else if assertionPassed {
		assertionWidget.Text = "Passing"
	} else {
		assertionWidget.Text = "Failing"
	}

	utils.UpdatePlot(uptimePlot, utils.BoolToFloat64(isUp), refreshFlag)
	utils.UpdatePlot(responseTimePlot, responseTime.Seconds(), refreshFlag)

	grid.SetRect(0, 0, width, height)

	ui.Render(grid)

	return lastCheckTime, isUp
}

func UpdateQuitWidgetText(newText string) {
	if quitWidget != nil {
		quitWidget.Text = newText
		ui.Render(quitWidget)
	}
}
