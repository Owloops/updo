package tui

import (
	"fmt"
	"image"
	"strings"
	"time"

	ui "github.com/Owloops/termui/v3"

	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
	"github.com/Owloops/updo/utils"
	uw "github.com/Owloops/updo/widgets"
)

const (
	_focusNone focusArea = iota
	_focusTargets
	_focusSearch
	_focusLogs
)

const (
	_historyCap        = 600
	_placeholder       = "—"
	_brand             = " updo "
	_httpsPrefix       = "https://"
	_localRegion       = "local"
	_uptimeGood        = 99.9
	_uptimeWarn        = 95.0
	_sslWarnDays       = 30
	_sslBadDays        = 7
	_sidebarPercent    = 26
	_sidebarMinWidth   = 24
	_sidebarMaxWidth   = 38
	_searchHeight      = 3
	_tilesHeight       = 4
	_lowerHeight       = 8
	_minChartHeight    = 5
	_minLogsHeight     = 7
	_logsPercent       = 30
	_pulsePercent      = 58
	_minPlaceWidth     = 4
	_percent           = 100
	_clockLayout       = "15:04:05"
	_microsPerMilli    = 1000.0
	_targetsHeading    = "Targets"
	_searchHeading     = "Search"
	_logsHeading       = "Logs"
	_responseHeading   = "Response time"
	_checksHeading     = "Checks"
	_timingHeading     = "Timing"
	_overviewHeading   = "Overview"
	_waitingForCheck   = "waiting for the first check"
	_noTimingData      = "no timing data for the last check"
	_logsExpandHint    = "enter expand"
	_statusUp          = " UP "
	_statusDown        = " DOWN "
	_statusPending     = " WAIT "
	_assertPass        = "pass"
	_assertFail        = "fail"
	_headerSeparator   = "  "
	_headerRegionArrow = " › "
)

type focusArea int

type screen struct {
	ui.Block
	draw func(buf *ui.Buffer, rect image.Rectangle)
}

type dashWidgets struct {
	header *uw.Bar
	keybar *uw.Bar
	tiles  *uw.StatTiles
	chart  *uw.AreaChart
	pulse  *uw.Pulse
	search *uw.Framed
	logs   *uw.Framed
}

type detailView struct {
	key       stats.TargetKey
	target    config.Target
	hasTarget bool
	hasResult bool
	result    net.WebsiteCheckResult
	stats     stats.Stats
	sslDays   int
}

type keyHint struct {
	key  string
	desc string
}

func newScreen(draw func(buf *ui.Buffer, rect image.Rectangle)) *screen {
	return &screen{Block: *ui.NewBlock(), draw: draw}
}

func (s *screen) Draw(buf *ui.Buffer) {
	s.draw(buf, s.GetRect())
}

func (m *Manager) initStyledWidgets() {
	d := m.detailsManager
	m.dash = &dashWidgets{
		header: uw.NewBar(),
		keybar: uw.NewBar(),
		tiles:  uw.NewStatTiles(),
		chart:  uw.NewAreaChart(),
		pulse:  uw.NewPulse(),
		logs:   uw.NewFramed(d.LogsWidget, &d.LogsWidget.Block),
	}
	if m.searchWidget != nil {
		m.dash.search = uw.NewFramed(m.searchWidget, &m.searchWidget.Block)
	}
}

func (m *Manager) focus() focusArea {
	switch {
	case m.listWidget != nil && m.listWidget.IsSearchMode():
		return _focusSearch
	case m.focusOnLogs:
		return _focusLogs
	case m.listWidget != nil:
		return _focusTargets
	}
	return _focusNone
}

func (m *Manager) currentView() detailView {
	var view detailView

	key := m.getCurrentTargetKey()
	if key == nil {
		if group := m.getKeysForCurrentSelection(); len(group) > 0 {
			key = &group[0]
		}
	}
	if key == nil {
		allKeys := m.keyRegistry.GetAllKeys()
		if m.currentKeyIndex < 0 || m.currentKeyIndex >= len(allKeys) {
			return view
		}
		key = &allKeys[m.currentKeyIndex]
	}
	view.key = *key

	if key.TargetIndex >= 0 && key.TargetIndex < len(m.targets) {
		view.target = m.targets[key.TargetIndex]
		view.hasTarget = true
	}

	if data, exists := m.targetData[key.String()]; exists {
		view.hasResult = true
		view.result = data.Result
		view.stats = data.Stats
	}
	if live, exists := m.liveStats[key.String()]; exists {
		view.stats = live
	}

	if view.hasTarget {
		view.sslDays = m.getSSLExpiry(view.target.URL)
	}

	return view
}

func (m *Manager) rememberSample(targetKey string, result net.WebsiteCheckResult) {
	history := m.history[targetKey]
	history = append(history, uw.Sample{
		Millis:      float64(result.ResponseTime.Microseconds()) / _microsPerMilli,
		IsUp:        result.IsUp,
		HasResponse: result.StatusCode > 0,
	})
	if len(history) > _historyCap {
		history = history[len(history)-_historyCap:]
	}
	m.history[targetKey] = history
}

func (m *Manager) drawDash(buf *ui.Buffer, rect image.Rectangle) {
	m.prepareDash()
	for _, item := range m.layoutDash(rect) {
		item.Lock()
		item.Draw(buf)
		item.Unlock()
	}
}

func (m *Manager) layoutDash(rect image.Rectangle) []ui.Drawable {
	var placed []ui.Drawable
	place := func(item ui.Drawable, x1, y1, x2, y2 int) {
		if x2-x1 < _minPlaceWidth || y2 <= y1 {
			return
		}
		item.SetRect(x1, y1, x2, y2)
		placed = append(placed, item)
	}

	d := m.dash
	place(d.header, rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+1)
	place(d.keybar, rect.Min.X, rect.Max.Y-1, rect.Max.X, rect.Max.Y)

	top, bottom := rect.Min.Y+1, rect.Max.Y-1

	x := rect.Min.X
	if m.listWidget != nil {
		sidebar := min(max(rect.Dx()*_sidebarPercent/_percent, _sidebarMinWidth), _sidebarMaxWidth)
		place(d.search, x, top, x+sidebar, top+_searchHeight)
		place(m.listWidget, x, top+_searchHeight, x+sidebar, bottom)
		x += sidebar
	}

	y := top
	place(d.tiles, x, y, rect.Max.X, y+_tilesHeight)
	y += _tilesHeight

	logsHeight := 0
	if m.showLogs {
		logsHeight = max((bottom-top)*_logsPercent/_percent, _minLogsHeight)
	}
	lowerHeight := _lowerHeight
	chartHeight := bottom - y - lowerHeight - logsHeight
	if chartHeight < _minChartHeight {
		lowerHeight = 0
		chartHeight = bottom - y - logsHeight
	}

	place(d.chart, x, y, rect.Max.X, y+chartHeight)
	y += max(chartHeight, 0)

	if lowerHeight > 0 {
		split := x + (rect.Max.X-x)*_pulsePercent/_percent
		place(d.pulse, x, y, split, y+lowerHeight)
		place(m.detailsManager.TimingBreakdownWidget, split, y, rect.Max.X, y+lowerHeight)
		y += lowerHeight
	}
	if logsHeight > 0 {
		place(d.logs, x, y, rect.Max.X, bottom)
	}

	return placed
}

func (m *Manager) prepareDash() {
	p, d, view, focus := m.pal, m.dash, m.currentView(), m.focus()

	frame := func(heading, aside string, isFocused bool) uw.Frame {
		f := uw.Frame{Heading: heading, HeadingStyle: p.muted, Aside: aside, AsideStyle: p.muted, LineStyle: p.faint}
		if isFocused {
			f.HeadingStyle = p.accent
			f.LineStyle = ui.NewStyle(p.accent.Fg)
		}
		return f
	}

	history := m.history[view.key.String()]

	d.header.Left, d.header.Right = m.headerSpans(view)
	d.keybar.Left = m.keyHintSpans(focus)

	d.tiles.Frame = frame(_overviewHeading, "", false)
	d.tiles.LabelStyle = p.muted
	d.tiles.Tiles = m.tiles(view)

	chartAside := ""
	if last := len(history) - 1; last >= 0 && (history[last].IsUp || history[last].HasResponse) {
		chartAside = "last " + uw.FormatMillis(history[len(history)-1].Millis)
	}
	d.chart.Frame = frame(_responseHeading, chartAside, false)
	d.chart.Samples = history
	d.chart.Interval = 0
	if view.hasTarget {
		d.chart.Interval = view.target.GetRefreshInterval()
	}
	d.chart.BarStyle, d.chart.FailStyle = ui.NewStyle(p.accent.Fg), ui.NewStyle(p.bad.Fg)
	d.chart.AxisStyle, d.chart.GridStyle = p.muted, p.faint
	d.chart.EmptyMessage = _waitingForCheck

	checksAside := ""
	if view.stats.ChecksCount > 0 {
		failed := view.stats.ChecksCount - view.stats.SuccessCount
		checksAside = fmt.Sprintf("%d ok · %d failed", view.stats.SuccessCount, failed)
	}
	d.pulse.Frame = frame(_checksHeading, checksAside, false)
	d.pulse.Beats = history
	d.pulse.Facts = m.facts(view)
	d.pulse.UpStyle, d.pulse.DownStyle = p.good, p.bad
	d.pulse.TrackStyle, d.pulse.LabelStyle = p.faint, p.muted

	timing := m.detailsManager.TimingBreakdownWidget
	timingAside := ""
	if view.hasResult && view.result.TraceInfo != nil {
		timingAside = formatDuration(view.result.ResponseTime)
	}
	timing.Frame = frame(_timingHeading, timingAside, false)
	timing.LabelStyle, timing.ValueStyle = p.muted, p.text
	timing.EmptyMessage = _noTimingData
	timing.SetTimings(stageTimings(view.result))

	logs := m.detailsManager.LogsWidget
	d.logs.Frame = frame(_logsHeading, _logsExpandHint, focus == _focusLogs)
	logs.TextStyle, logs.SelectedRowStyle = p.text, p.selected

	if m.listWidget == nil {
		return
	}

	listAside := fmt.Sprintf("%d", len(m.keyRegistry.GetAllKeys()))
	if focus == _focusSearch && m.listWidget.GetQuery() != "" {
		listAside = fmt.Sprintf("%d of %s", m.matchingKeyCount(), listAside)
	}
	m.listWidget.Frame = frame(_targetsHeading, listAside, focus == _focusTargets)
	m.listWidget.Look = uw.ListLook{
		Text:     p.text,
		Header:   p.strong,
		Chevron:  p.faint,
		Marker:   ui.NewStyle(p.accent.Fg),
		Spark:    p.faint,
		Selected: p.selected,
	}

	d.search.Frame = frame(_searchHeading, "", focus == _focusSearch)
	m.searchWidget.TextStyle = p.muted
	if focus == _focusSearch && m.listWidget.GetQuery() != "" {
		m.searchWidget.TextStyle = p.strong
	}
}

func (m *Manager) matchingKeyCount() int {
	count := 0
	for _, idx := range m.listWidget.GetFilteredIndices() {
		if idx < len(m.itemToKeyIndex) && m.itemToKeyIndex[idx] >= 0 {
			count++
		}
	}
	return count
}

func (m *Manager) headerSpans(view detailView) (left, right []uw.Span) {
	p := m.pal
	gap := uw.Span{Text: _headerSeparator, Style: p.text}

	left = []uw.Span{{Text: _brand, Style: p.accentBar}, gap}
	if view.hasTarget {
		left = append(left, uw.Span{Text: view.key.GetCleanName(), Style: p.strong})
		if !view.key.IsLocal && view.key.Region != "" {
			left = append(left, uw.Span{Text: _headerRegionArrow + view.key.Region, Style: p.muted})
		}
		left = append(left, gap)
	}

	switch {
	case !view.hasResult:
		left = append(left, uw.Span{Text: _statusPending, Style: pill(p.warn)})
	case view.result.IsUp:
		left = append(left, uw.Span{Text: _statusUp, Style: pill(p.good)})
	default:
		left = append(left, uw.Span{Text: _statusDown, Style: pill(p.bad)})
	}
	if view.hasResult && view.result.StatusCode > 0 {
		left = append(left, uw.Span{Text: fmt.Sprintf(" %d", view.result.StatusCode), Style: p.muted})
	}
	if view.hasTarget {
		left = append(left, gap, uw.Span{Text: view.target.URL, Style: p.muted})
	}

	if m.isSingle {
		return left, nil
	}

	up, down, pending := 0, 0, 0
	for _, key := range m.keyRegistry.GetAllKeys() {
		data, exists := m.targetData[key.String()]
		switch {
		case !exists:
			pending++
		case data.Result.IsUp:
			up++
		default:
			down++
		}
	}
	count := func(icon ui.Style, format string, n int) []uw.Span {
		return []uw.Span{{Text: _targetIcon + " ", Style: icon}, {Text: fmt.Sprintf(format, n), Style: p.text}}
	}
	right = count(p.good, "%d up", up)
	if down > 0 {
		right = append(append(right, gap), count(p.bad, "%d down", down)...)
	}
	if pending > 0 {
		right = append(append(right, gap), count(p.warn, "%d waiting", pending)...)
	}
	return left, right
}

func (m *Manager) keyHintSpans(focus focusArea) []uw.Span {
	var hints []keyHint
	switch focus {
	case _focusSearch:
		hints = []keyHint{{"type", "filter"}, {"↑↓", "navigate"}, {"esc", "close"}}
	case _focusLogs:
		hints = []keyHint{{"↑↓", "scroll"}, {"enter", "expand"}, {"l", "close logs"}}
	case _focusTargets:
		hints = []keyHint{{"↑↓", "navigate"}, {"enter", "fold"}, {"tab", "fold all"}, {"/", "search"}, {"l", "logs"}}
	default:
		hints = []keyHint{{"l", "logs"}}
	}
	if focus != _focusSearch {
		hints = append(hints, keyHint{"q", "quit"})
	}

	spans := make([]uw.Span, 0, len(hints)*2)
	for _, hint := range hints {
		spans = append(spans,
			uw.Span{Text: hint.key, Style: m.pal.accent},
			uw.Span{Text: " " + hint.desc + _headerSeparator, Style: m.pal.muted},
		)
	}
	return spans
}

func (m *Manager) tiles(view detailView) []uw.Tile {
	p := m.pal
	figure := func(label string, d time.Duration, isReady bool) uw.Tile {
		if !isReady {
			return uw.Tile{Label: label, Value: _placeholder, ValueStyle: p.muted}
		}
		return uw.Tile{Label: label, Value: formatDuration(d), ValueStyle: p.strong}
	}

	hasChecks := view.stats.ChecksCount > 0
	uptime := uw.Tile{Label: "UPTIME", Value: _placeholder, ValueStyle: uptimeStyle(p, view)}
	if hasChecks {
		uptime.Value = fmt.Sprintf("%.2f%%", view.stats.UptimePercent)
	}

	return []uw.Tile{
		uptime,
		figure("AVG", view.stats.AvgResponseTime, hasChecks),
		figure("P95", view.stats.P95, view.stats.ChecksCount >= 2),
		figure("MIN", view.stats.MinResponseTime, hasChecks),
		figure("MAX", view.stats.MaxResponseTime, hasChecks),
		{Label: "SSL", Value: sslText(view), ValueStyle: sslStyle(p, view)},
		{Label: "ASSERT", Value: assertionText(view), ValueStyle: assertionStyle(p, view)},
	}
}

func (m *Manager) facts(view detailView) []uw.Fact {
	p := m.pal
	fact := func(label, value string) uw.Fact {
		if value == "" {
			return uw.Fact{Label: label, Value: _placeholder, ValueStyle: p.muted}
		}
		return uw.Fact{Label: label, Value: value, ValueStyle: p.text}
	}

	lastCheck, address := "", ""
	if view.hasResult {
		address = view.result.ResolvedIP
		if !view.result.LastCheckTime.IsZero() {
			checkedAt := view.result.LastCheckTime
			lastCheck = fmt.Sprintf("%s · %s ago", checkedAt.Format(_clockLayout), time.Since(checkedAt).Truncate(time.Second))
		}
	}
	if address == "" {
		address = view.stats.LastIP
	}

	interval := ""
	if view.hasTarget {
		interval = "every " + view.target.GetRefreshInterval().String()
	}
	running := ""
	if view.stats.TotalDuration > 0 {
		running = utils.FormatDurationMinute(view.stats.TotalDuration)
	}

	return []uw.Fact{
		fact("Last check", lastCheck),
		fact("Address", address),
		fact("Interval", interval),
		fact("Watching", running),
	}
}

func uptimeStyle(p palette, view detailView) ui.Style {
	switch {
	case view.stats.ChecksCount == 0:
		return p.muted
	case view.stats.UptimePercent >= _uptimeGood:
		return p.good
	case view.stats.UptimePercent >= _uptimeWarn:
		return p.warn
	}
	return p.bad
}

func assertionText(view detailView) string {
	switch {
	case !view.hasResult || view.result.AssertText == "":
		return _placeholder
	case view.result.AssertionPassed:
		return _assertPass
	}
	return _assertFail
}

func assertionStyle(p palette, view detailView) ui.Style {
	switch assertionText(view) {
	case _assertPass:
		return p.good
	case _assertFail:
		return p.bad
	}
	return p.muted
}

func sslText(view detailView) string {
	if view.sslDays <= 0 {
		return _placeholder
	}
	return fmt.Sprintf("%dd", view.sslDays)
}

func sslStyle(p palette, view detailView) ui.Style {
	switch {
	case !strings.HasPrefix(view.target.URL, _httpsPrefix) || view.sslDays <= 0:
		return p.muted
	case view.sslDays > _sslWarnDays:
		return p.good
	case view.sslDays > _sslBadDays:
		return p.warn
	}
	return p.bad
}

func formatDuration(d time.Duration) string {
	return uw.FormatMillis(float64(d.Microseconds()) / _microsPerMilli)
}
