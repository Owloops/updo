package widgets

import (
	ui "github.com/Owloops/termui/v3"
)

// Framed swaps a stock termui widget's square border and title for a Frame.
type Framed struct {
	ui.Block
	Frame     Frame
	Body      ui.Drawable
	BodyBlock *ui.Block
}

func NewFramed(body ui.Drawable, bodyBlock *ui.Block) *Framed {
	return &Framed{Block: *ui.NewBlock(), Body: body, BodyBlock: bodyBlock}
}

func (f *Framed) Draw(buf *ui.Buffer) {
	rect := f.GetRect()
	inner := f.Frame.Inner(rect)
	if !inner.Empty() {
		f.Body.Lock()
		border, title := f.BodyBlock.Border, f.BodyBlock.Title
		f.BodyBlock.Border, f.BodyBlock.Title = false, ""
		f.Body.SetRect(inner.Min.X-1, inner.Min.Y-1, inner.Max.X+1, inner.Max.Y+1)
		f.Body.Draw(buf)
		f.BodyBlock.Border, f.BodyBlock.Title = border, title
		f.Body.Unlock()
	}
	f.Frame.Draw(buf, rect)
}
