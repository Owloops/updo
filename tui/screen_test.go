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

func newTestManager(t *testing.T, names ...string) *Manager {
	t.Helper()

	targets := make([]config.Target, 0, len(names))
	for _, name := range names {
		targets = append(targets, config.Target{Name: name, URL: "https://" + name + ".test"})
	}

	m := NewManager(targets, Options{})
	for _, target := range targets {
		m.sslExpiry[target.URL] = 0
	}
	m.detailsManager.InitializeWidgets()
	if !m.isSingle {
		m.initializeMultiTargetWidgets()
	}
	m.initStyledWidgets()
	return m
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
			m := newTestManager(t, names...)
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
	m := newTestManager(t, "one", "two")

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
	if got := rowString(buf, 1); !containsAll(got, "Search", "Overview") {
		t.Errorf("panel headings = %q", got)
	}
}

func TestTargetListIsFlatForLocalTargets(t *testing.T) {
	m := newTestManager(t, "one", "two")
	m.updateTargetList()

	if got := len(m.listWidget.Rows); got != 2 {
		t.Fatalf("rows = %d (%q), want one row per local target", got, m.listWidget.Rows)
	}
	for i, name := range []string{"one", "two"} {
		if row := m.listWidget.Rows[i]; !strings.HasSuffix(row, " "+name) || m.listWidget.IsHeaderAtIndex(i) {
			t.Errorf("row %d = %q, want a selectable %q target row", i, row, name)
		}
	}

	m.listWidget.SelectedRow = 1
	if key := m.getCurrentTargetKey(); key == nil || key.GetCleanName() != "two" {
		t.Errorf("selected key = %v, want two", key)
	}
}

func TestTargetListGroupsRegions(t *testing.T) {
	targets := []config.Target{
		{Name: "one", URL: "https://one.test"},
		{Name: "two", URL: "https://two.test", Regions: []string{"us-east-1", "eu-west-1"}},
	}
	m := NewManager(targets, Options{})
	for _, target := range targets {
		m.sslExpiry[target.URL] = 0
	}
	m.detailsManager.InitializeWidgets()
	m.initializeMultiTargetWidgets()
	m.initStyledWidgets()
	m.updateTargetList()

	rows := m.listWidget.Rows
	if len(rows) < 4 || !m.listWidget.IsHeaderAtIndex(1) || m.listWidget.IsHeaderAtIndex(0) {
		t.Fatalf("rows = %q, want a flat row then a region group", rows)
	}
	if !containsAll(strings.Join(rows[2:], "|"), "us-east-1", "eu-west-1") {
		t.Errorf("region rows = %q", rows[2:])
	}

	m.listWidget.SelectedRow = 1
	if view := m.currentView(); view.key.GetCleanName() != "two" || view.key.Region != "us-east-1" {
		t.Errorf("group header shows %v, want the first region of two", view.key)
	}
}

func TestRememberSampleCapsHistory(t *testing.T) {
	m := newTestManager(t, "solo")
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
