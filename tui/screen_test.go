package tui

import (
	"image"
	"strings"
	"testing"
	"time"

	ui "github.com/Owloops/termui/v3"

	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
)

func newStyledManager(t *testing.T, style uiStyle, names ...string) *Manager {
	t.Helper()

	targets := make([]config.Target, 0, len(names))
	for _, name := range names {
		targets = append(targets, config.Target{Name: name, URL: "https://" + name + ".test"})
	}

	m := NewManager(targets, Options{})
	m.style = style
	for _, target := range targets {
		m.sslExpiry[target.URL] = 0
	}
	first := targets[0]
	m.detailsManager.createWidgets(first.URL, first.GetRefreshInterval())
	if !m.isSingle {
		m.initializeMultiTargetWidgets()
	}
	m.initStyledWidgets()
	return m
}

func TestStyleFromEnv(t *testing.T) {
	tests := []struct {
		value string
		want  uiStyle
	}{
		{"", _styleDash},
		{" Classic ", _styleClassic},
		{"nonsense", _styleDash},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv(_styleEnv, tt.value)
			if got := styleFromEnv(); got != tt.want {
				t.Errorf("styleFromEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccentFromEnv(t *testing.T) {
	tests := []struct {
		value string
		want  ui.Color
	}{
		{"", _colorBrandOrange},
		{" Cyan ", ui.ColorCyan},
		{"nope", _colorBrandOrange},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv(_accentEnv, tt.value)
			if got := accentFromEnv(); got != tt.want {
				t.Errorf("accentFromEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLayoutDashStaysInsideAndDoesNotOverlap(t *testing.T) {
	sizes := []image.Point{{140, 46}, {80, 24}, {60, 14}, {30, 8}, {200, 60}}

	for _, names := range [][]string{{"solo"}, {"one", "two", "three"}} {
		for _, showLogs := range []bool{false, true} {
			m := newStyledManager(t, _styleDash, names...)
			m.showLogs = showLogs

			for _, size := range sizes {
				screenRect := image.Rect(0, 0, size.X, size.Y)
				var seen []image.Rectangle
				for _, item := range m.layoutDash(screenRect) {
					rect := item.GetRect()
					if !rect.In(screenRect) {
						t.Errorf("%v logs=%v: %v outside the screen", size, showLogs, rect)
					}
					for _, other := range seen {
						if rect.Overlaps(other) {
							t.Errorf("%v logs=%v: %v overlaps %v", size, showLogs, rect, other)
						}
					}
					seen = append(seen, rect)
				}
			}
		}
	}
}

func TestDrawDash(t *testing.T) {
	m := newStyledManager(t, _styleDash, "one", "two")

	key := m.keyRegistry.GetAllKeys()[0]
	result := net.WebsiteCheckResult{
		URL:          "https://one.test",
		IsUp:         true,
		StatusCode:   200,
		ResponseTime: 120 * time.Millisecond,
	}
	m.targetData[key.String()] = TargetData{
		Target:    m.targets[0],
		Result:    result,
		TargetKey: key,
		Stats:     stats.Stats{ChecksCount: 1, SuccessCount: 1, UptimePercent: 100},
	}
	m.rememberSample(key.String(), result)
	m.updateTargetList()

	rect := image.Rect(0, 0, 120, 40)
	buf := ui.NewBuffer(rect)
	m.drawDash(buf, rect)
	if got := rowString(buf, 0); !containsAll(got, _brand, "one", _statusUp, "https://one.test") {
		t.Errorf("header = %q", got)
	}
	if m.listWidget.Frame == nil || m.detailsManager.TimingBreakdownWidget.Frame == nil {
		t.Fatal("dash did not frame the shared widgets")
	}
}

func TestRememberSampleCapsHistory(t *testing.T) {
	m := newStyledManager(t, _styleDash, "solo")
	for range _historyCap + 25 {
		m.rememberSample("solo", net.WebsiteCheckResult{IsUp: true, ResponseTime: time.Millisecond})
	}
	if got := len(m.history["solo"]); got != _historyCap {
		t.Errorf("history length = %d, want %d", got, _historyCap)
	}
}

func rowString(buf *ui.Buffer, y int) string {
	row := make([]rune, 0, buf.Dx())
	for x := buf.Min.X; x < buf.Max.X; x++ {
		row = append(row, buf.GetCell(image.Pt(x, y)).Rune)
	}
	return string(row)
}

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
