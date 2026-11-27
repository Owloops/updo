package widgets

import (
	ui "github.com/Owloops/termui/v3"
)

// SearchBox shows the search text above a note about the result.
type SearchBox struct {
	ui.Block
	Frame     Frame
	Text      string
	TextStyle ui.Style
	Note      string
	NoteStyle ui.Style
}

func NewSearchBox() *SearchBox {
	return &SearchBox{Block: *ui.NewBlock()}
}

func (sb *SearchBox) Draw(buf *ui.Buffer) {
	rect := sb.GetRect()
	sb.Frame.Draw(buf, rect)
	inner := sb.Frame.Inner(rect)
	if inner.Empty() {
		return
	}

	DrawText(buf, sb.Text, sb.TextStyle, inner.Min.X, inner.Min.Y, inner.Max.X)
	if inner.Dy() > 1 {
		DrawText(buf, sb.Note, sb.NoteStyle, inner.Min.X, inner.Min.Y+1, inner.Max.X)
	}
}
