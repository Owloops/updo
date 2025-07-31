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
				ui.NewCol(0.22,
					ui.NewRow(1.0/7, m.searchWidget),
					ui.NewRow(6.0/7, m.listWidget),
				),
				ui.NewCol(0.78, m.detailsManager.ActiveGrid),
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
	allKeys := m.keyRegistry.GetAllKeys()

	m.searchWidget = widgets.NewParagraph()
	m.searchWidget.Border = true
	m.searchWidget.BorderStyle.Fg = ui.ColorCyan
	m.searchWidget.Title = "Search"
	m.searchWidget.TitleStyle.Fg = ui.ColorWhite
	m.searchWidget.TitleStyle.Modifier = ui.ModifierBold
	m.searchWidget.Text = "Press / to activate"
	m.searchWidget.TextStyle.Fg = ui.ColorWhite

	m.listWidget = uw.NewFilteredList()
	if len(allKeys) > 0 {
		m.listWidget.Title = fmt.Sprintf("Targets → %s", allKeys[0].DisplayName())
	} else {
		m.listWidget.Title = "Targets"
	}
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
			m.listWidget.Title = fmt.Sprintf("Targets (%d/%d)", len(filteredIndices), len(allKeys))
		} else {
			m.searchWidget.Text = "Type to filter..."
			m.searchWidget.TextStyle.Fg = ui.ColorYellow
			m.searchWidget.BorderStyle.Fg = ui.ColorYellow
			m.listWidget.Title = "Targets"
		}

		keyVisible := false
		if len(filteredIndices) > 0 {
			for i, idx := range filteredIndices {
				if idx == m.currentKeyIndex {
					m.listWidget.SelectedRow = i
					keyVisible = true
					break
				}
			}
		}

		if !keyVisible {
			m.listWidget.SelectedRow = 0
			m.listWidget.SelectedRowStyle = m.listWidget.TextStyle
		}
	} else {
		m.searchWidget.Text = "Press / to activate"
		m.searchWidget.TextStyle.Fg = ui.ColorWhite
		m.searchWidget.BorderStyle.Fg = ui.ColorCyan
		if m.currentKeyIndex < len(allKeys) {
			m.listWidget.Title = fmt.Sprintf("Targets → %s", allKeys[m.currentKeyIndex].DisplayName())
		}
		m.listWidget.SelectedRow = m.currentKeyIndex
	}
}

func (m *Manager) NavigateTargetKeys(direction int, monitors map[string]*stats.Monitor) {
	if m.listWidget == nil {
		return
	}

	allKeys := m.keyRegistry.GetAllKeys()
	if len(allKeys) == 0 {
		return
	}

	if m.listWidget.IsSearchMode() {
		m.navigateFilteredKeys(direction, allKeys)
	} else {
		m.navigateAllKeys(direction, allKeys)
	}

	m.updateActiveTarget(monitors)
}

func (m *Manager) navigateFilteredKeys(direction int, _ []TargetKey) {
	filteredIndices := m.listWidget.GetFilteredIndices()
	if len(filteredIndices) == 0 {
		return
	}

	currentFilteredIndex := -1
	for i, idx := range filteredIndices {
		if idx == m.currentKeyIndex {
			currentFilteredIndex = i
			break
		}
	}

	if currentFilteredIndex == -1 {
		if direction > 0 {
			currentFilteredIndex = 0
		} else {
			currentFilteredIndex = len(filteredIndices) - 1
		}
	} else {
		if direction > 0 {
			currentFilteredIndex = (currentFilteredIndex + 1) % len(filteredIndices)
		} else {
			currentFilteredIndex = (currentFilteredIndex - 1 + len(filteredIndices)) % len(filteredIndices)
		}
	}

	m.currentKeyIndex = filteredIndices[currentFilteredIndex]
	m.listWidget.SelectedRow = currentFilteredIndex
}

func (m *Manager) navigateAllKeys(direction int, allKeys []TargetKey) {
	if direction > 0 {
		m.currentKeyIndex = (m.currentKeyIndex + 1) % len(allKeys)
	} else {
		m.currentKeyIndex = (m.currentKeyIndex - 1 + len(allKeys)) % len(allKeys)
	}
	m.listWidget.SelectedRow = m.currentKeyIndex
}
