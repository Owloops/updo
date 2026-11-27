package widgets

import (
	"fmt"
	"image"
	"time"

	ui "github.com/Owloops/termui/v3"
	rw "github.com/mattn/go-runewidth"
)

const (
	_maxDurationWidth     = len(" 9999ms")
	_labelWidth           = 9
	_durationFormat       = "%4dms"
	_labelFormat          = "%-9s"
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

var (
	_stagesOrder = []string{StageWait, StageDNS, StageTCP, StageTLS, StageTTFB, StageDownload}
	_labelStyle  = ui.NewStyle(ui.ColorWhite)
)

type TimingBreakdown struct {
	ui.Block
	Timings map[string]time.Duration
	Colors  map[string]ui.Color

	Frame      *Frame
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
	if tb.Frame != nil {
		tb.drawWaterfall(buf)
		return
	}

	tb.Block.Draw(buf)

	totalDuration := tb.calculateTotalDuration()
	if totalDuration == 0 {
		return
	}

	y := tb.Inner.Min.Y
	longestLabel := tb.longestLabel()

	for _, stage := range _stagesOrder {
		duration, ok := tb.Timings[stage]
		if !ok {
			continue
		}

		percentage := float64(duration) / float64(totalDuration)
		barWidth := int(percentage * float64(tb.Inner.Dx()-longestLabel-_maxDurationWidth))

		label := fmt.Sprintf(_labelFormat, stage)
		roundedDuration := fmt.Sprintf(_durationFormat, int64(duration.Seconds()*_millisecondsInSecond))

		for i, r := range label {
			buf.SetCell(ui.NewCell(r, _labelStyle), image.Pt(tb.Inner.Min.X+i, y))
		}

		for i, r := range roundedDuration {
			buf.SetCell(ui.NewCell(r, _labelStyle), image.Pt(tb.Inner.Min.X+longestLabel+i, y))
		}

		xStart := tb.Inner.Min.X + longestLabel + rw.StringWidth(roundedDuration) + 1
		for i := range barWidth {
			buf.SetCell(ui.NewCell(' ', ui.NewStyle(ui.ColorClear, tb.Colors[stage])), image.Pt(xStart+i, y))
		}
		y++
	}
}

func (tb *TimingBreakdown) drawWaterfall(buf *ui.Buffer) {
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

func (tb *TimingBreakdown) longestLabel() int {
	return _labelWidth + 1
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
