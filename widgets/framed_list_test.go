package widgets

import (
	"image"
	"strings"
	"testing"
	"time"

	ui "github.com/Owloops/termui/v3"
)

func TestFilteredListFramedDraw(t *testing.T) {
	selectedBg := ui.Color(237)
	fl := NewFilteredList()
	fl.Frame = &Frame{Heading: "Targets"}
	fl.Look = ListLook{Selected: ui.NewStyle(ui.ColorWhite, selectedBg)}
	fl.SetRowsWithMetadata(
		[]string{"▼ GitHub", "    [◉](fg:green) local"},
		[]RowMetadata{
			{GroupID: "GitHub", IsHeader: true, IsSelectable: true},
			{GroupID: "GitHub", IsSelectable: true, Right: "42ms"},
		},
	)
	fl.SelectedRow = 1
	fl.SetRect(0, 0, 24, 4)
	buf := ui.NewBuffer(fl.GetRect())
	fl.Draw(buf)

	if got := rowText(buf, 1); !strings.Contains(got, "▼ GitHub") {
		t.Errorf("header row = %q", got)
	}
	row := rowText(buf, 2)
	hasLatency := strings.HasSuffix(strings.TrimRight(row, " │"), "42ms")
	if !strings.Contains(row, "▌") || !strings.Contains(row, "◉ local") || !hasLatency {
		t.Errorf("selected row = %q", row)
	}
	if strings.Contains(row, "fg:green") {
		t.Errorf("style markup leaked into %q", row)
	}

	for x := 2; x < 22; x++ {
		if got := buf.GetCell(image.Pt(x, 2)).Style.Bg; got != selectedBg {
			t.Fatalf("selection background missing at x=%d", x)
		}
	}
	if got := buf.GetCell(image.Pt(6, 2)).Style.Fg; got != ui.ColorGreen {
		t.Errorf("status icon colour = %v, want green kept under selection", got)
	}
}

func TestFilteredListFramedScrollsToSelection(t *testing.T) {
	fl := NewFilteredList()
	fl.Frame = &Frame{}
	fl.SetRows([]string{"one", "two", "three", "four", "five"})
	fl.SelectedRow = 4
	fl.SetRect(0, 0, 16, 4)
	buf := ui.NewBuffer(fl.GetRect())
	fl.Draw(buf)

	if got := rowText(buf, 2); !strings.Contains(got, "five") {
		t.Errorf("last visible row = %q, want the selected row", got)
	}
}

func TestTimingBreakdownWaterfall(t *testing.T) {
	tb := NewTimingBreakdown()
	tb.Frame = &Frame{Heading: "Timing"}
	tb.EmptyMessage = "no timing data"
	tb.SetRect(0, 0, 40, 8)

	buf := ui.NewBuffer(tb.GetRect())
	tb.Draw(buf)
	if got := rowText(buf, 1); !strings.Contains(got, "no timing data") {
		t.Errorf("empty row = %q", got)
	}

	tb.SetTimings(map[string]time.Duration{
		StageWait:     20 * time.Millisecond,
		StageDNS:      10 * time.Millisecond,
		StageTCP:      30 * time.Millisecond,
		StageTLS:      40 * time.Millisecond,
		StageTTFB:     60 * time.Millisecond,
		StageDownload: 40 * time.Millisecond,
	})
	buf = ui.NewBuffer(tb.GetRect())
	tb.Draw(buf)

	previousEnd := 0
	for y := 1; y <= len(_stagesOrder); y++ {
		row := []rune(rowText(buf, y))
		start, end := -1, -1
		for x, r := range row {
			if r == _waterfallRune {
				if start < 0 {
					start = x
				}
				end = x + 1
			}
		}
		if start < 0 {
			t.Fatalf("row %d has no bar: %q", y, string(row))
		}
		if start < previousEnd-1 {
			t.Errorf("row %d starts at %d, before the previous stage ended at %d", y, start, previousEnd)
		}
		previousEnd = end
	}
}
