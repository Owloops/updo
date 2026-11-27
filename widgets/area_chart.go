package widgets

import (
	"fmt"
	"image"
	"math"
	"time"

	ui "github.com/Owloops/termui/v3"
	rw "github.com/mattn/go-runewidth"
)

const (
	_axisWidth          = 7
	_maxBarWidth        = 3
	_minRowsForMidTick  = 5
	_msPerSecond        = 1000.0
	_sparkPerRune       = 2
	_sparkLevels        = 4
	_sparkFlatRatio     = 0.15
	_brailleBase        = 0x2800
	_minRowsForTimeAxis = 5
	_nowLabel           = "now"
)

var (
	_brailleDots = [_sparkPerRune][_sparkLevels]rune{{0x40, 0x04, 0x02, 0x01}, {0x80, 0x20, 0x10, 0x08}}
	_niceSteps   = [...]float64{1, 1.5, 2, 3, 4, 5, 7.5, 10}
	_fill        = []rune{' ', '⣀', '⣤', '⣶', '⣿'}
)

// Sample is the outcome of one check.
type Sample struct {
	Millis      float64
	IsUp        bool
	HasResponse bool
}

// AreaChart is a filled, right-anchored history chart of response times above a time axis.
type AreaChart struct {
	ui.Block
	Frame        Frame
	Samples      []Sample
	Interval     time.Duration
	BarStyle     ui.Style
	FailStyle    ui.Style
	AxisStyle    ui.Style
	GridStyle    ui.Style
	EmptyMessage string
}

func NewAreaChart() *AreaChart {
	return &AreaChart{Block: *ui.NewBlock()}
}

func (c *AreaChart) Draw(buf *ui.Buffer) {
	rect := c.GetRect()
	c.Frame.Draw(buf, rect)
	inner := c.Frame.Inner(rect)
	if inner.Empty() {
		return
	}

	plot := image.Rect(inner.Min.X+_axisWidth, inner.Min.Y, inner.Max.X, inner.Max.Y)
	if plot.Dx() <= 0 {
		return
	}
	hasTimeAxis := inner.Dy() >= _minRowsForTimeAxis
	if hasTimeAxis {
		plot.Max.Y--
	}

	if len(c.Samples) == 0 {
		x := inner.Min.X + max((inner.Dx()-rw.StringWidth(c.EmptyMessage))/2, 0)
		DrawText(buf, c.EmptyMessage, c.AxisStyle, x, inner.Min.Y+inner.Dy()/2, inner.Max.X)
		return
	}

	width := barWidth(len(c.Samples), plot.Dx())
	columns := plot.Dx() / width
	visible := c.Samples[max(len(c.Samples)-columns, 0):]
	highest := maxMillis(visible)
	ceiling := NiceCeil(highest)
	rows := plot.Dy()

	if highest > 0 {
		c.drawAxis(buf, inner, plot, ceiling)
	}

	steps := len(_fill) - 1
	filled := max(width-1, 1)
	left := plot.Max.X - len(visible)*width
	for i, sample := range visible {
		x := left + i*width
		style := c.BarStyle
		if !sample.IsUp {
			if !sample.HasResponse {
				continue
			}
			style = c.FailStyle
		}
		level := int(math.Round(sample.Millis / ceiling * float64(rows*steps)))
		for dx := range filled {
			drawColumn(buf, x+dx, plot.Max.Y-1, min(max(level, 1), rows*steps), style)
		}
	}

	if hasTimeAxis {
		c.drawTimeAxis(buf, plot, columns)
	}
}

func (c *AreaChart) drawTimeAxis(buf *ui.Buffer, plot image.Rectangle, columns int) {
	y := plot.Max.Y
	DrawTextRight(buf, _nowLabel, c.AxisStyle, plot.Min.X, y, plot.Max.X)
	if c.Interval > 0 {
		span := "-" + FormatSpan(time.Duration(columns)*c.Interval)
		DrawText(buf, span, c.AxisStyle, plot.Min.X, y, plot.Max.X-len(_nowLabel)-1)
	}
}

func (c *AreaChart) drawAxis(buf *ui.Buffer, inner, plot image.Rectangle, ceiling float64) {
	ticks := map[int]float64{0: ceiling}
	if plot.Dy() >= _minRowsForMidTick {
		ticks[plot.Dy()/2] = ceiling * float64(plot.Dy()-plot.Dy()/2) / float64(plot.Dy())
	}
	for row, value := range ticks {
		y := plot.Min.Y + row
		buf.Fill(ui.NewCell(ui.HORIZONTAL_DASH, c.GridStyle), image.Rect(plot.Min.X, y, plot.Max.X, y+1))
		DrawTextRight(buf, FormatMillis(value), c.AxisStyle, inner.Min.X, y, plot.Min.X-1)
	}
	if _, isTaken := ticks[plot.Dy()-1]; !isTaken {
		DrawTextRight(buf, "0", c.AxisStyle, inner.Min.X, plot.Max.Y-1, plot.Min.X-1)
	}
}

// barWidth widens bars while history is short so a new chart is not mostly empty.
func barWidth(samples, columns int) int {
	return min(max(columns/samples, 1), _maxBarWidth)
}

func drawColumn(buf *ui.Buffer, x, bottomY, level int, style ui.Style) {
	steps := len(_fill) - 1
	full, partial := level/steps, level%steps
	for row := range full {
		buf.SetCell(ui.NewCell(_fill[steps], style), image.Pt(x, bottomY-row))
	}
	if partial > 0 {
		buf.SetCell(ui.NewCell(_fill[partial], style), image.Pt(x, bottomY-full))
	}
}

func maxMillis(samples []Sample) float64 {
	highest := 0.0
	for _, sample := range samples {
		if sample.IsUp || sample.HasResponse {
			highest = math.Max(highest, sample.Millis)
		}
	}
	return highest
}

// NiceCeil rounds v up to a human-friendly axis maximum.
func NiceCeil(v float64) float64 {
	if v <= 1 {
		return 1
	}
	magnitude := math.Pow(10, math.Floor(math.Log10(v)))
	for _, step := range _niceSteps {
		if candidate := step * magnitude; candidate >= v {
			return candidate
		}
	}
	return v
}

// FormatMillis renders a millisecond value compactly, switching to seconds past 10s.
func FormatMillis(ms float64) string {
	if ms >= 10*_msPerSecond {
		return fmt.Sprintf("%.1fs", ms/_msPerSecond)
	}
	return fmt.Sprintf("%.0fms", ms)
}

// FormatSpan renders a duration compactly with at most two units, such as 45s, 4m30s or 2h05m.
func FormatSpan(d time.Duration) string {
	d = d.Round(time.Second)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		if seconds := int(d.Seconds()) % 60; seconds != 0 {
			return fmt.Sprintf("%dm%02ds", int(d.Minutes()), seconds)
		}
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}

// Sparkline renders the newest samples as a one-row braille area that is width runes wide.
func Sparkline(samples []Sample, width int) []rune {
	out := make([]rune, width)
	for i := range out {
		out[i] = ' '
	}
	samples = samples[max(len(samples)-width*_sparkPerRune, 0):]
	if len(samples) == 0 {
		return out
	}

	lowest, highest := math.Inf(1), 0.0
	for _, sample := range samples {
		if sample.IsUp {
			lowest, highest = math.Min(lowest, sample.Millis), math.Max(highest, sample.Millis)
		}
	}
	isFlat := highest-lowest <= highest*_sparkFlatRatio

	offset := width*_sparkPerRune - len(samples)
	for i, sample := range samples {
		if !sample.IsUp {
			continue
		}
		level := 1
		if !isFlat {
			level += int(math.Round((sample.Millis - lowest) / (highest - lowest) * (_sparkLevels - 1)))
		}
		cell, side := (offset+i)/_sparkPerRune, (offset+i)%_sparkPerRune
		if out[cell] == ' ' {
			out[cell] = _brailleBase
		}
		for dot := range level {
			out[cell] |= _brailleDots[side][dot]
		}
	}
	return out
}
