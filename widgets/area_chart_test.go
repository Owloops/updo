package widgets

import (
	"image"
	"slices"
	"testing"
	"time"

	ui "github.com/Owloops/termui/v3"
)

func TestNiceCeil(t *testing.T) {
	tests := []struct {
		in, want float64
	}{
		{0, 1}, {0.4, 1}, {1, 1}, {87, 100}, {101, 150}, {420, 500}, {957, 1000}, {1442, 1500}, {1714, 2000}, {6100, 7500},
	}

	for _, tt := range tests {
		if got := NiceCeil(tt.in); got != tt.want {
			t.Errorf("NiceCeil(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestFormatMillis(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0ms"}, {0.6, "1ms"}, {541.2, "541ms"}, {9999, "9999ms"}, {12500, "12.5s"},
	}

	for _, tt := range tests {
		if got := FormatMillis(tt.in); got != tt.want {
			t.Errorf("FormatMillis(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatSpan(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{45 * time.Second, "45s"}, {2 * time.Minute, "2m"}, {270 * time.Second, "4m30s"}, {125 * time.Minute, "2h05m"},
	}

	for _, tt := range tests {
		if got := FormatSpan(tt.in); got != tt.want {
			t.Errorf("FormatSpan(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSparkline(t *testing.T) {
	ok := func(ms float64) Sample { return Sample{Millis: ms, IsUp: true} }
	tests := []struct {
		name    string
		samples []Sample
		want    string
	}{
		{"empty is blank", nil, "    "},
		{"steady series sits on the baseline", []Sample{ok(100), ok(101), ok(100), ok(102)}, "  ⣀⣀"},
		{"spikes rise and the newest is rightmost", []Sample{ok(100), ok(400)}, "   ⣸"},
		{"failed checks leave a gap", []Sample{ok(100), {Millis: 5000}, ok(100)}, "  ⢀⢀"},
		{"only the newest samples are kept", slices.Repeat([]Sample{ok(1)}, 10), "⣀⣀⣀⣀"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(Sparkline(tt.samples, 4)); got != tt.want {
				t.Errorf("Sparkline() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAreaChartDraw(t *testing.T) {
	up := Sample{Millis: 100, IsUp: true}
	tests := []struct {
		name    string
		height  int
		samples []Sample
		rows    map[int]string
	}{
		{
			name:   "empty shows the message",
			height: 7,
			rows:   map[int]string{3: "│     waiting      │"},
		},
		{
			name:    "newest sample is right-anchored and scaled to a nice ceiling",
			height:  7,
			samples: []Sample{{Millis: 50, IsUp: true}, up},
			rows: map[int]string{
				1: "│  100ms ┈┈┈┈┈███┈ │",
				4: "│      0  ███ ███  │",
				5: "│        -4s   now │",
			},
		},
		{
			name:    "failed checks without a response leave a gap and do not set the scale",
			height:  7,
			samples: []Sample{{Millis: 9000}, {Millis: 40, IsUp: true}},
			rows: map[int]string{
				1: "│   40ms ┈┈┈┈┈███┈ │",
				4: "│      0      ███  │",
			},
		},
		{
			name:    "failed checks with a response keep their bar and set the scale",
			height:  7,
			samples: []Sample{{Millis: 200, HasResponse: true}, {Millis: 40, IsUp: true}},
			rows: map[int]string{
				1: "│  200ms ┈███┈┈┈┈┈ │",
				4: "│      0  ███ ▆▆▆  │",
			},
		},
		{
			name:    "a full history keeps a gap between bars and drops the oldest checks",
			height:  7,
			samples: []Sample{up, up, up, {Millis: 100}, up, up},
			rows: map[int]string{
				4: "│      0  █   █ █  │",
				5: "│        -8s   now │",
			},
		},
		{
			name:    "short charts drop the time axis",
			height:  6,
			samples: []Sample{up},
			rows:    map[int]string{4: "│      0      ███  │"},
		},
		{
			name:    "every check has a beat as wide as its bar",
			height:  11,
			samples: []Sample{up, {Millis: 100}, up},
			rows: map[int]string{
				6: "│      0 ██    ██  │",
				7: "│                  │",
				8: "│ checks ██ ██ ██  │",
				9: "│        -6s   now │",
			},
		},
		{
			name:    "tall charts double the beat row",
			height:  14,
			samples: []Sample{up, {Millis: 100}, up},
			rows: map[int]string{
				10: "│ checks ██ ██ ██  │",
				11: "│        ██ ██ ██  │",
				12: "│        -6s   now │",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chart := NewAreaChart()
			chart.Samples = tt.samples
			chart.Interval = 2 * time.Second
			chart.EmptyMessage = "waiting"
			chart.BarGradient = []ui.Style{ui.NewStyle(ui.ColorBlue)}
			chart.SetRect(0, 0, 20, tt.height)
			buf := ui.NewBuffer(chart.GetRect())
			chart.Draw(buf)

			for y, want := range tt.rows {
				if got := rowText(buf, y); got != want {
					t.Errorf("row %d = %q, want %q", y, got, want)
				}
			}
		})
	}
}

func TestAreaChartStyles(t *testing.T) {
	chart := NewAreaChart()
	chart.Samples = []Sample{{Millis: 100, IsUp: true}, {Millis: 100, HasResponse: true}, {Millis: 100, IsUp: true}}
	chart.BarGradient = []ui.Style{ui.NewStyle(ui.ColorBlue), ui.NewStyle(ui.ColorCyan)}
	chart.FailStyle = ui.NewStyle(ui.ColorMagenta)
	chart.UpStyle, chart.DownStyle = ui.NewStyle(ui.ColorGreen), ui.NewStyle(ui.ColorRed)
	chart.PendingStyle = ui.NewStyle(ui.ColorWhite)
	chart.SetRect(0, 0, 27, 11)
	buf := ui.NewBuffer(chart.GetRect())
	chart.Draw(buf)

	tests := []struct {
		name string
		at   image.Point
		want ui.Color
	}{
		{"bar starts with the first gradient colour", image.Pt(21, 6), ui.ColorBlue},
		{"bar ends with the last gradient colour", image.Pt(21, 1), ui.ColorCyan},
		{"failed bar", image.Pt(17, 6), ui.ColorMagenta},
		{"up beat", image.Pt(21, 8), ui.ColorGreen},
		{"down beat", image.Pt(17, 8), ui.ColorRed},
		{"pending beat", image.Pt(9, 8), ui.ColorWhite},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buf.GetCell(tt.at).Style.Fg; got != tt.want {
				t.Errorf("colour at %v = %v, want %v", tt.at, got, tt.want)
			}
		})
	}
}

func TestAreaChartTinyRectDoesNotPanic(t *testing.T) {
	chart := NewAreaChart()
	chart.Samples = []Sample{{Millis: 1, IsUp: true}}
	chart.SetRect(0, 0, 3, 1)
	chart.Draw(ui.NewBuffer(chart.GetRect()))
}
