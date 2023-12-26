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

func (self *TimingBreakdown) Draw(buf *ui.Buffer) {
	self.Block.Draw(buf)

	totalDuration := self.calculateTotalDuration()
	if totalDuration == 0 {
		return
	}

	y := self.Inner.Min.Y
	longestLabel := self.longestLabel()

	for _, stage := range stagesOrder {
		duration, ok := self.Timings[stage]
		if !ok {
			continue
		}

		percentage := float64(duration) / float64(totalDuration)
		barWidth := int(percentage * float64(self.Inner.Dx()-longestLabel-rw.StringWidth(" 9999ms")))

		label := fmt.Sprintf("%-9s", stage)
		roundedDuration := fmt.Sprintf("%4dms", int64(duration.Seconds()*1000))
		labelStyle := ui.NewStyle(ui.ColorWhite)

		for i, rune := range label {
			buf.SetCell(ui.NewCell(rune, labelStyle), image.Pt(self.Inner.Min.X+i, y))
		}

		for i, rune := range roundedDuration {
			buf.SetCell(ui.NewCell(rune, labelStyle), image.Pt(self.Inner.Min.X+longestLabel+i, y))
		}

		xStart := self.Inner.Min.X + longestLabel + rw.StringWidth(roundedDuration) + 1
		for i := 0; i < barWidth; i++ {
			buf.SetCell(ui.NewCell(' ', ui.NewStyle(ui.ColorClear, self.Colors[stage])), image.Pt(xStart+i, y))
		}
		y++
	}
}

func (self *TimingBreakdown) longestLabel() int {
	longest := 0
	for _, stage := range stagesOrder {
		if len(stage) > longest {
			longest = len(stage)
		}
	}
	return longest + 1
}

func (self *TimingBreakdown) calculateTotalDuration() time.Duration {
	var total time.Duration
	for _, duration := range self.Timings {
		total += duration
	}
	return total
}

func (self *TimingBreakdown) SetTimings(newTimings map[string]time.Duration) {
	self.Timings = newTimings
}
