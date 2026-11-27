package widgets

import (
	ui "github.com/Owloops/termui/v3"
)

const (
	_minTileWidth = 10
	_tileGap      = 2
)

type Tile struct {
	Label      string
	Value      string
	ValueStyle ui.Style
}

// StatTiles is a strip of label-over-value figures.
type StatTiles struct {
	ui.Block
	Frame      Frame
	Tiles      []Tile
	LabelStyle ui.Style
}

func NewStatTiles() *StatTiles {
	return &StatTiles{Block: *ui.NewBlock()}
}

func (st *StatTiles) Draw(buf *ui.Buffer) {
	rect := st.GetRect()
	st.Frame.Draw(buf, rect)
	inner := st.Frame.Inner(rect)
	if inner.Empty() || len(st.Tiles) == 0 {
		return
	}

	count := min(len(st.Tiles), max(inner.Dx()/_minTileWidth, 1))
	width := inner.Dx() / count

	for i, tile := range st.Tiles[:count] {
		x := inner.Min.X + i*width
		limit := x + width - _tileGap
		if inner.Dy() == 1 {
			DrawText(buf, tile.Value, tile.ValueStyle, x, inner.Min.Y, limit)
			continue
		}
		DrawText(buf, tile.Label, st.LabelStyle, x, inner.Min.Y, limit)
		DrawText(buf, tile.Value, tile.ValueStyle, x, inner.Min.Y+1, limit)
	}
}
