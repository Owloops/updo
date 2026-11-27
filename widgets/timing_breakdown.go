package widgets

import (
	"image"
	"time"

	ui "github.com/Owloops/termui/v3"
)

const (
	_millisecondsInSecond = 1000
	_waterfallRune        = '▄'
	_waterfallLabelWidth  = 10
	_waterfallValueWidth  = 7

	StageWait     = "Wait"
	StageDNS      = "DNS"
	StageTCP      = "TCP"
	StageTLS      = "TLS"
	StageTTFB     = "TTFB"
	StageDownload = "Download"
)

var _stagesOrder = []string{StageWait, StageDNS, StageTCP, StageTLS, StageTTFB, StageDownload}

type TimingBreakdown struct {
	ui.Block
	Timings map[string]time.Duration
	Colors  map[string]ui.Color

	Frame      Frame
	LabelStyle ui.Style
	ValueStyle ui.Style

	EmptyMessage string
}

func NewTimingBreakdown() *TimingBreakdown {
	return &TimingBreakdown{
		Block:   *ui.NewBlock(),
		Timings: make(map[string]time.Duration),
		Colors: map[string]ui.Color{
			StageWait:     ui.ColorCyan,
			StageDNS:      ui.ColorYellow,
			StageTCP:      ui.ColorRed,
			StageTLS:      ui.ColorMagenta,
			StageTTFB:     ui.ColorBlue,
			StageDownload: ui.ColorGreen,
		},
	}
}

func (tb *TimingBreakdown) Draw(buf *ui.Buffer) {
	rect := tb.GetRect()
	tb.Frame.Draw(buf, rect)
	inner := tb.Frame.Inner(rect)
	totalDuration := tb.calculateTotalDuration()
	if inner.Empty() {
		return
	}
	if totalDuration == 0 {
		DrawText(buf, tb.EmptyMessage, tb.LabelStyle, inner.Min.X, inner.Min.Y, inner.Max.X)
		return
	}

	trackStart := inner.Min.X + _waterfallLabelWidth
	trackEnd := inner.Max.X - _waterfallValueWidth
	trackWidth := trackEnd - trackStart

	var elapsed time.Duration
	y := inner.Min.Y
	for _, stage := range _stagesOrder {
		duration, ok := tb.Timings[stage]
		if !ok {
			continue
		}
		if y >= inner.Max.Y {
			break
		}

		DrawText(buf, stage, tb.LabelStyle, inner.Min.X, y, trackStart-1)
		value := FormatMillis(float64(duration.Microseconds()) / _millisecondsInSecond)
		DrawTextRight(buf, value, tb.ValueStyle, trackEnd, y, inner.Max.X)

		if trackWidth > 0 {
			start := int(float64(elapsed) / float64(totalDuration) * float64(trackWidth))
			end := int(float64(elapsed+duration) / float64(totalDuration) * float64(trackWidth))
			if duration > 0 {
				start = min(start, trackWidth-1)
				end = max(end, start+1)
			}
			bar := ui.NewCell(_waterfallRune, ui.NewStyle(tb.Colors[stage]))
			buf.Fill(bar, image.Rect(trackStart+start, y, trackStart+end, y+1))
		}

		elapsed += duration
		y++
	}
}

func (tb *TimingBreakdown) calculateTotalDuration() time.Duration {
	var total time.Duration
	for _, duration := range tb.Timings {
		total += duration
	}
	return total
}

func (tb *TimingBreakdown) SetTimings(newTimings map[string]time.Duration) {
	tb.Timings = newTimings
}
