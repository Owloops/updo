package widgets

import (
	"image"

	ui "github.com/Owloops/termui/v3"
	rw "github.com/mattn/go-runewidth"
)

const (
	_roundTopLeft     = '╭'
	_roundTopRight    = '╮'
	_roundBottomLeft  = '╰'
	_roundBottomRight = '╯'
	_frameMinSize     = 2
	_boxPadding       = 2
)

// Frame draws the chrome around a widget: a rounded box with a heading and an aside in its top edge.
type Frame struct {
	Heading      string
	HeadingStyle ui.Style
	Aside        string
	AsideStyle   ui.Style
	LineStyle    ui.Style
}

// Inner returns the content area left inside r once the chrome is drawn.
func (f *Frame) Inner(r image.Rectangle) image.Rectangle {
	if r.Dx() <= 2*_boxPadding || r.Dy() <= _frameMinSize {
		return image.Rectangle{}
	}
	return image.Rect(r.Min.X+_boxPadding, r.Min.Y+1, r.Max.X-_boxPadding, r.Max.Y-1)
}

// Draw renders the chrome onto buf within r.
func (f *Frame) Draw(buf *ui.Buffer, r image.Rectangle) {
	if r.Dx() < _frameMinSize || r.Dy() < _frameMinSize {
		return
	}

	left, right := r.Min.X+_boxPadding, r.Max.X-_boxPadding
	f.drawBox(buf, r)

	headingEnd := left
	if f.Heading != "" {
		headingEnd = DrawText(buf, " "+f.Heading+" ", f.HeadingStyle, left, r.Min.Y, right)
	}

	if f.Aside == "" {
		return
	}
	aside := " " + f.Aside + " "
	asideStart := right - rw.StringWidth(aside)
	if asideStart > headingEnd {
		DrawText(buf, aside, f.AsideStyle, asideStart, r.Min.Y, right)
	}
}

func (f *Frame) drawBox(buf *ui.Buffer, r image.Rectangle) {
	horizontal := ui.NewCell(ui.HORIZONTAL_LINE, f.LineStyle)
	vertical := ui.NewCell(ui.VERTICAL_LINE, f.LineStyle)

	buf.Fill(horizontal, image.Rect(r.Min.X+1, r.Min.Y, r.Max.X-1, r.Min.Y+1))
	buf.Fill(horizontal, image.Rect(r.Min.X+1, r.Max.Y-1, r.Max.X-1, r.Max.Y))
	buf.Fill(vertical, image.Rect(r.Min.X, r.Min.Y+1, r.Min.X+1, r.Max.Y-1))
	buf.Fill(vertical, image.Rect(r.Max.X-1, r.Min.Y+1, r.Max.X, r.Max.Y-1))

	buf.SetCell(ui.NewCell(_roundTopLeft, f.LineStyle), r.Min)
	buf.SetCell(ui.NewCell(_roundTopRight, f.LineStyle), image.Pt(r.Max.X-1, r.Min.Y))
	buf.SetCell(ui.NewCell(_roundBottomLeft, f.LineStyle), image.Pt(r.Min.X, r.Max.Y-1))
	buf.SetCell(ui.NewCell(_roundBottomRight, f.LineStyle), image.Pt(r.Max.X-1, r.Max.Y-1))
}

// DrawText writes s at (x, y), truncating with an ellipsis at maxX, and returns the next free column.
func DrawText(buf *ui.Buffer, s string, style ui.Style, x, y, maxX int) int {
	available := maxX - x
	if available <= 0 || s == "" {
		return x
	}
	if rw.StringWidth(s) > available {
		s = rw.Truncate(s, available, string(ui.ELLIPSES))
	}
	for _, r := range s {
		buf.SetCell(ui.NewCell(r, style), image.Pt(x, y))
		x += rw.RuneWidth(r)
	}
	return x
}

// DrawTextRight writes s so that it ends at maxX, never starting before minX.
func DrawTextRight(buf *ui.Buffer, s string, style ui.Style, minX, y, maxX int) {
	start := max(maxX-rw.StringWidth(s), minX)
	DrawText(buf, s, style, start, y, maxX)
}
