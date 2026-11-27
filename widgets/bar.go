package widgets

import (
	ui "github.com/Owloops/termui/v3"
	rw "github.com/mattn/go-runewidth"
)

const _barGap = 2

type Span struct {
	Text  string
	Style ui.Style
}

// Bar is a single borderless line with left and right aligned styled spans.
type Bar struct {
	ui.Block
	Left  []Span
	Right []Span
}

func NewBar() *Bar {
	bar := &Bar{Block: *ui.NewBlock()}
	bar.Border = false
	return bar
}

func (b *Bar) Draw(buf *ui.Buffer) {
	rect := b.GetRect()
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	minX, maxX, y := rect.Min.X+1, rect.Max.X-1, rect.Min.Y

	leftLimit := maxX
	rightWidth := spansWidth(b.Right)
	if rightWidth > 0 && spansWidth(b.Left)+_barGap+rightWidth <= maxX-minX {
		x := maxX - rightWidth
		leftLimit = x - _barGap
		for _, span := range b.Right {
			x = DrawText(buf, span.Text, span.Style, x, y, maxX)
		}
	}

	x := minX
	for _, span := range b.Left {
		x = DrawText(buf, span.Text, span.Style, x, y, leftLimit)
	}
}

func spansWidth(spans []Span) int {
	width := 0
	for _, span := range spans {
		width += rw.StringWidth(span.Text)
	}
	return width
}
