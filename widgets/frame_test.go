package widgets

import (
	"image"
	"strings"
	"testing"

	ui "github.com/Owloops/termui/v3"
)

func rowText(buf *ui.Buffer, y int) string {
	var row strings.Builder
	for x := buf.Min.X; x < buf.Max.X; x++ {
		row.WriteRune(buf.GetCell(image.Pt(x, y)).Rune)
	}
	return row.String()
}

func TestFrameInner(t *testing.T) {
	tests := []struct {
		name string
		rect image.Rectangle
		want image.Rectangle
	}{
		{"pads content off the border", image.Rect(0, 0, 20, 6), image.Rect(2, 1, 18, 5)},
		{"too small collapses to empty", image.Rect(0, 0, 3, 2), image.Rectangle{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := Frame{}
			if got := frame.Inner(tt.rect); got != tt.want {
				t.Errorf("Inner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFrameDraw(t *testing.T) {
	rect := image.Rect(0, 0, 24, 3)
	buf := ui.NewBuffer(rect)
	frame := Frame{Heading: "Checks", Aside: "3 ok"}
	frame.Draw(buf, rect)
	if got, want := rowText(buf, 0), "╭─ Checks ────── 3 ok ─╮"; got != want {
		t.Errorf("top row = %q, want %q", got, want)
	}
	if got, want := rowText(buf, 2), "╰──────────────────────╯"; got != want {
		t.Errorf("bottom row = %q, want %q", got, want)
	}
}

func TestDrawTextTruncates(t *testing.T) {
	rect := image.Rect(0, 0, 10, 1)
	buf := ui.NewBuffer(rect)
	next := DrawText(buf, "response time", ui.StyleClear, 0, 0, 6)

	if got := rowText(buf, 0); got != "respo…    " {
		t.Errorf("row = %q", got)
	}
	if next != 6 {
		t.Errorf("next column = %d, want 6", next)
	}
}
