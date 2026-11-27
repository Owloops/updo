package widgets

import (
	"image"

	ui "github.com/Owloops/termui/v3"
	rw "github.com/mattn/go-runewidth"
)

const (
	_beatRune      = '┃'
	_trackRune     = '·'
	_factsOffset   = 2
	_factLabelGap  = 2
	_minRowsForGap = 4
)

type Fact struct {
	Label      string
	Value      string
	ValueStyle ui.Style
}

// Pulse shows one tick per check, newest on the right, above a short list of facts.
type Pulse struct {
	ui.Block
	Frame      Frame
	Beats      []Sample
	Facts      []Fact
	UpStyle    ui.Style
	DownStyle  ui.Style
	TrackStyle ui.Style
	LabelStyle ui.Style
}

func NewPulse() *Pulse {
	return &Pulse{Block: *ui.NewBlock()}
}

func (p *Pulse) Draw(buf *ui.Buffer) {
	rect := p.GetRect()
	p.Frame.Draw(buf, rect)
	inner := p.Frame.Inner(rect)
	if inner.Empty() {
		return
	}

	buf.Fill(ui.NewCell(_trackRune, p.TrackStyle), image.Rect(inner.Min.X, inner.Min.Y, inner.Max.X, inner.Min.Y+1))
	visible := p.Beats[max(len(p.Beats)-inner.Dx(), 0):]
	x := inner.Max.X - len(visible)
	for i, beat := range visible {
		style := p.UpStyle
		if !beat.IsUp {
			style = p.DownStyle
		}
		buf.SetCell(ui.NewCell(_beatRune, style), image.Pt(x+i, inner.Min.Y))
	}

	labelWidth := 0
	for _, fact := range p.Facts {
		labelWidth = max(labelWidth, rw.StringWidth(fact.Label))
	}

	y := inner.Min.Y + 1
	if inner.Dy() >= _minRowsForGap {
		y = inner.Min.Y + _factsOffset
	}
	for _, fact := range p.Facts {
		if y >= inner.Max.Y {
			break
		}
		DrawText(buf, fact.Label, p.LabelStyle, inner.Min.X, y, inner.Max.X)
		DrawText(buf, fact.Value, fact.ValueStyle, inner.Min.X+labelWidth+_factLabelGap, y, inner.Max.X)
		y++
	}
}
