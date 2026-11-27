package widgets

import (
	"testing"

	ui "github.com/Owloops/termui/v3"
)

func TestSearchBoxDraw(t *testing.T) {
	tests := []struct {
		name   string
		height int
		rows   map[int]string
	}{
		{
			name:   "text above the note",
			height: 4,
			rows: map[int]string{
				0: "╭─ Search ─────────╮",
				1: "│ git              │",
				2: "│ 1 of 5 match     │",
			},
		},
		{
			name:   "one row keeps only the text",
			height: 3,
			rows: map[int]string{
				1: "│ git              │",
				2: "╰──────────────────╯",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			box := NewSearchBox()
			box.Frame = Frame{Heading: "Search"}
			box.Text, box.Note = "git", "1 of 5 match"
			box.SetRect(0, 0, 20, tt.height)
			buf := ui.NewBuffer(box.GetRect())
			box.Draw(buf)

			for y, want := range tt.rows {
				if got := rowText(buf, y); got != want {
					t.Errorf("row %d = %q, want %q", y, got, want)
				}
			}
		})
	}
}
