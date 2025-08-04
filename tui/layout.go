package tui

import (
	"fmt"

	"github.com/Owloops/updo/stats"
	uw "github.com/Owloops/updo/widgets"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

func (m *Manager) setupGrid(width, height int) {
	m.grid = ui.NewGrid()
	m.grid.SetRect(0, 0, width, height)

	if m.isSingle {
		m.grid.Set(
			ui.NewRow(1.0, m.detailsManager.ActiveGrid),
		)
	} else {
		m.grid.Set(
			ui.NewRow(1.0,
				ui.NewCol(0.28,
					ui.NewRow(1.0/7, m.searchWidget),
					ui.NewRow(6.0/7, m.listWidget),
				),
				ui.NewCol(0.72, m.detailsManager.ActiveGrid),
			),
		)
	}

	ui.Render(m.grid)
}

func (m *Manager) Resize(width, height int) {
	m.termWidth = width
	m.termHeight = height
	m.grid.SetRect(0, 0, width, height)
}

func (m *Manager) initializeMultiTargetWidgets() {
	m.searchWidget = widgets.NewParagraph()
	m.searchWidget.Border = true
	m.searchWidget.BorderStyle.Fg = ui.ColorCyan
	m.searchWidget.Title = "Search"
	m.searchWidget.TitleStyle.Fg = ui.ColorWhite
	m.searchWidget.TitleStyle.Modifier = ui.ModifierBold
	m.searchWidget.Text = "Press / to activate"
	m.searchWidget.TextStyle.Fg = ui.ColorWhite

	m.listWidget = uw.NewFilteredList()
	m.listWidget.Title = _targetsTitle
	m.listWidget.BorderStyle.Fg = ui.ColorCyan
	m.listWidget.TitleStyle.Fg = ui.ColorWhite
	m.listWidget.TitleStyle.Modifier = ui.ModifierBold

	m.listWidget.OnSearchChange = func(query string, filteredIndices []int) {
		m.handleSearchChange(query, filteredIndices)
	}

	m.updateTargetList()
}

func (m *Manager) handleSearchChange(query string, filteredIndices []int) {
	allKeys := m.keyRegistry.GetAllKeys()

	if m.listWidget.IsSearchMode() {
		if query != "" {
			m.searchWidget.Text = query
			m.searchWidget.TextStyle.Fg = ui.ColorGreen
			m.searchWidget.BorderStyle.Fg = ui.ColorGreen
			filteredKeyCount := 0
			for _, idx := range filteredIndices {
				if idx < len(m.itemToKeyIndex) && m.itemToKeyIndex[idx] >= 0 {
					filteredKeyCount++
				}
			}
			m.listWidget.Title = fmt.Sprintf("Targets (%d/%d)", filteredKeyCount, len(allKeys))
		} else {
			m.searchWidget.Text = "Type to filter..."
			m.searchWidget.TextStyle.Fg = ui.ColorYellow
			m.searchWidget.BorderStyle.Fg = ui.ColorYellow
			m.listWidget.Title = _targetsTitle
		}

		visibleRows := len(m.listWidget.Rows)
		if visibleRows > 0 {
			if m.listWidget.SelectedRow >= visibleRows || m.listWidget.SelectedRow < 0 {
				m.listWidget.SelectedRow = 0
			}
		}
	} else {
		m.searchWidget.Text = "Press / to activate"
		m.searchWidget.TextStyle.Fg = ui.ColorWhite
		m.searchWidget.BorderStyle.Fg = ui.ColorCyan

		currentKey := m.getCurrentTargetKey()
		if currentKey != nil {
			m.listWidget.Title = fmt.Sprintf("Targets → %s", currentKey.GetCleanName())
		} else {
			m.listWidget.Title = _targetsTitle
		}
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
		m.updateSelectionColors()
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
		if !m.isSingle {
			m.updateSelectionColors()
		}
		ui.Render(m.grid)
	}
}
