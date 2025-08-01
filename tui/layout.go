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
	m.listWidget.Title = "Targets"
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
			displayMap := m.listWidget.GetFilteredDisplayIndices()

			for itemIdx, keyIdx := range m.itemToKeyIndex {
				if keyIdx == m.currentKeyIndex {
					if displayIdx, ok := displayMap[itemIdx]; ok {
						m.listWidget.SelectedRow = displayIdx
						keyVisible = true
						break
					}
				}
			}
		}

		if !keyVisible && len(filteredIndices) > 0 {
			selectableIndices := m.listWidget.GetSelectableIndices()
			if len(selectableIndices) > 0 {
				firstSelectableIdx := selectableIndices[0]
				if firstSelectableIdx < len(m.itemToKeyIndex) {
					m.currentKeyIndex = m.itemToKeyIndex[firstSelectableIdx]
				}
				displayMap := m.listWidget.GetFilteredDisplayIndices()
				if displayIdx, ok := displayMap[firstSelectableIdx]; ok {
					m.listWidget.SelectedRow = displayIdx
				}
			}
		}
	} else {
		m.searchWidget.Text = "Press / to activate"
		m.searchWidget.TextStyle.Fg = ui.ColorWhite
		m.searchWidget.BorderStyle.Fg = ui.ColorCyan
		if m.currentKeyIndex < len(allKeys) {
			currentKey := allKeys[m.currentKeyIndex]
			region := "local"
			if !currentKey.IsLocal && currentKey.Region != "" {
				region = currentKey.Region
			}
			m.listWidget.Title = fmt.Sprintf("Targets → %s [%s]", currentKey.TargetName, region)
		}
		for i, keyIdx := range m.itemToKeyIndex {
			if keyIdx == m.currentKeyIndex {
				m.listWidget.SelectedRow = i
				break
			}
		}
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
	selectableIndices := m.listWidget.GetSelectableIndices()
	if len(selectableIndices) == 0 {
		return
	}

	currentPos := -1
	for i, itemIdx := range selectableIndices {
		if itemIdx < len(m.itemToKeyIndex) && m.itemToKeyIndex[itemIdx] == m.currentKeyIndex {
			currentPos = i
			break
		}
	}

	if currentPos == -1 {
		currentPos = 0
	} else {
		if direction > 0 {
			currentPos = (currentPos + 1) % len(selectableIndices)
		} else {
			currentPos = (currentPos - 1 + len(selectableIndices)) % len(selectableIndices)
		}
	}

	selectedItemIdx := selectableIndices[currentPos]

	if selectedItemIdx < len(m.itemToKeyIndex) {
		newKeyIdx := m.itemToKeyIndex[selectedItemIdx]
		if newKeyIdx >= 0 {
			m.currentKeyIndex = newKeyIdx
		}
	}

	displayMap := m.listWidget.GetFilteredDisplayIndices()
	if displayIdx, ok := displayMap[selectedItemIdx]; ok {
		m.listWidget.SelectedRow = displayIdx
	}
}

func (m *Manager) navigateAllKeys(direction int, allKeys []TargetKey) {
	if direction > 0 {
		m.currentKeyIndex = (m.currentKeyIndex + 1) % len(allKeys)
	} else {
		m.currentKeyIndex = (m.currentKeyIndex - 1 + len(allKeys)) % len(allKeys)
	}

	for i, keyIdx := range m.itemToKeyIndex {
		if keyIdx == m.currentKeyIndex {
			m.listWidget.SelectedRow = i
			break
		}
	}
}
