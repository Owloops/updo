package widgets

import (
	"fmt"
	"image"
	"time"

	ui "github.com/gizak/termui/v3"
	rw "github.com/mattn/go-runewidth"
)

var stagesOrder = []string{"Wait", "DNS", "TCP", "TTFB", "Download"}

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

	for _, stage := range stagesOrder {
		duration, ok := tb.Timings[stage]
		if !ok {
			continue
		}

		percentage := float64(duration) / float64(totalDuration)
		barWidth := int(percentage * float64(tb.Inner.Dx()-longestLabel-rw.StringWidth(" 9999ms")))

		label := fmt.Sprintf("%-9s", stage)
		roundedDuration := fmt.Sprintf("%4dms", int64(duration.Seconds()*1000))
		labelStyle := ui.NewStyle(ui.ColorWhite)

		for i, rune := range label {
			buf.SetCell(ui.NewCell(rune, labelStyle), image.Pt(tb.Inner.Min.X+i, y))
		}

		for i, rune := range roundedDuration {
			buf.SetCell(ui.NewCell(rune, labelStyle), image.Pt(tb.Inner.Min.X+longestLabel+i, y))
		}

		xStart := tb.Inner.Min.X + longestLabel + rw.StringWidth(roundedDuration) + 1
		for i := 0; i < barWidth; i++ {
			buf.SetCell(ui.NewCell(' ', ui.NewStyle(ui.ColorClear, tb.Colors[stage])), image.Pt(xStart+i, y))
		}
		y++
	}
}

func (tb *TimingBreakdown) longestLabel() int {
	longest := 0
	for _, stage := range stagesOrder {
		if len(stage) > longest {
			longest = len(stage)
		}
	}
	return longest + 1
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
