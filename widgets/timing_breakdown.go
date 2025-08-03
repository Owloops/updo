package widgets

import (
	"fmt"
	"image"
	"time"

	ui "github.com/gizak/termui/v3"
	rw "github.com/mattn/go-runewidth"
)

const (
	_maxDurationWidth     = len(" 9999ms")
	_labelWidth           = 9
	_durationFormat       = "%4dms"
	_labelFormat          = "%-9s"
	_millisecondsInSecond = 1000
)

var (
	_stagesOrder = []string{"Wait", "DNS", "TCP", "TTFB", "Download"}
	_labelStyle  = ui.NewStyle(ui.ColorWhite)
)

type TimingBreakdown struct {
	ui.Block
	Timings map[string]time.Duration
	Colors  map[string]ui.Color
}

func NewTimingBreakdown() *TimingBreakdown {
	return &TimingBreakdown{
		Block:   *ui.NewBlock(),
		Timings: make(map[string]time.Duration),
		Colors: map[string]ui.Color{
			"Wait":     ui.ColorCyan,
			"DNS":      ui.ColorYellow,
			"TCP":      ui.ColorRed,
			"TTFB":     ui.ColorBlue,
			"Download": ui.ColorGreen,
		},
	}
}

func (tb *TimingBreakdown) Draw(buf *ui.Buffer) {
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
