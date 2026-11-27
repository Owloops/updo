package tui

import (
	ui "github.com/Owloops/termui/v3"
	"github.com/Owloops/termui/v3/widgets"
	"github.com/Owloops/updo/stats"
	uw "github.com/Owloops/updo/widgets"
)

const (
	_searchIdleHint   = "Press / to activate"
	_searchActiveHint = "Type to filter..."
)

func (m *Manager) setupGrid(width, height int) {
	m.grid = newScreen(m.drawDash)
	m.grid.SetRect(0, 0, width, height)

	ui.Render(m.grid)
}

func (m *Manager) Resize(width, height int) {
	m.termWidth = width
	m.termHeight = height
	m.grid.SetRect(0, 0, width, height)
}

func (m *Manager) initializeMultiTargetWidgets() {
	m.searchWidget = widgets.NewParagraph()
	m.searchWidget.Text = _searchIdleHint

	m.listWidget = uw.NewFilteredList()
	m.listWidget.OnSearchChange = func(query string, _ []int) {
		m.handleSearchChange(query)
	}

	m.updateTargetList()
}

func (m *Manager) handleSearchChange(query string) {
	switch {
	case !m.listWidget.IsSearchMode():
		m.searchWidget.Text = _searchIdleHint
	case query == "":
		m.searchWidget.Text = _searchActiveHint
	default:
		m.searchWidget.Text = query
	}

	if m.listWidget.SelectedRow >= len(m.listWidget.Rows) || m.listWidget.SelectedRow < 0 {
		m.listWidget.SelectedRow = 0
	}
}

func (m *Manager) NavigateTargetKeys(direction int, monitors map[string]*stats.Monitor) {
	if m.listWidget == nil {
		return
	}

	visibleRows := len(m.listWidget.Rows)
	if visibleRows == 0 {
		return
	}

	prevTargetKey := m.getCurrentTargetKey()
	prevKeyStr := ""
	if prevTargetKey != nil {
		prevKeyStr = prevTargetKey.String()
	}

	currentPos := m.listWidget.SelectedRow
	if direction > 0 {
		currentPos = (currentPos + 1) % visibleRows
	} else {
		currentPos = (currentPos - 1 + visibleRows) % visibleRows
	}
	m.listWidget.SelectedRow = currentPos

	if m.isSelectedRowHeader() {
		ui.Render(m.grid)
		return
	}

	newTargetKey := m.getCurrentTargetKey()
	if newTargetKey != nil {
		allKeys := m.keyRegistry.GetAllKeys()
		for i, key := range allKeys {
			if key.String() == newTargetKey.String() {
				m.currentKeyIndex = i
				break
			}
		}
	}

	newKeyStr := ""
	if newTargetKey != nil {
		newKeyStr = newTargetKey.String()
	}

	if newKeyStr != prevKeyStr {
		m.updateActiveTarget(monitors)
	} else {
		ui.Render(m.grid)
	}
}
