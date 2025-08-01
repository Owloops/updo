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
			m.listWidget.Title = "Targets"
		}

		visibleRows := len(m.listWidget.Rows)
		if visibleRows > 0 {
			currentRowValid := false
			currentRow := m.listWidget.SelectedRow

			if currentRow >= 0 && currentRow < len(filteredIndices) {
				originalIdx := filteredIndices[currentRow]
				if originalIdx < len(m.itemToKeyIndex) {
					keyIdx := m.itemToKeyIndex[originalIdx]
					if keyIdx == m.currentKeyIndex {
						currentRowValid = true
					}
				}
			}

			if !currentRowValid {
				for displayIdx, originalIdx := range filteredIndices {
					if originalIdx < len(m.itemToKeyIndex) {
						keyIdx := m.itemToKeyIndex[originalIdx]
						if keyIdx == m.currentKeyIndex {
							m.listWidget.SelectedRow = displayIdx
							currentRowValid = true
							break
						}
					}
				}
			}

			if !currentRowValid && visibleRows > 0 {
				m.listWidget.SelectedRow = 0
				if len(filteredIndices) > 0 {
					originalIdx := filteredIndices[0]
					if originalIdx < len(m.itemToKeyIndex) {
						keyIdx := m.itemToKeyIndex[originalIdx]
						if keyIdx >= 0 {
							m.currentKeyIndex = keyIdx
						}
					}
				}
			}
		}
	} else {
		m.searchWidget.Text = "Press / to activate"
		m.searchWidget.TextStyle.Fg = ui.ColorWhite
		m.searchWidget.BorderStyle.Fg = ui.ColorCyan
		if m.currentKeyIndex >= 0 && m.currentKeyIndex < len(allKeys) {
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

	previousKeyIndex := m.currentKeyIndex

	if m.listWidget.IsSearchMode() {
		m.navigateFilteredKeys(direction, allKeys)
	} else {
		m.navigateAllKeys(direction, allKeys)
	}

	if m.currentKeyIndex != previousKeyIndex {
		m.updateActiveTarget(monitors)
	} else {
		if !m.isSingle && !m.listWidget.IsSearchMode() {
			m.updateTargetList()
		}
		ui.Render(m.grid)
	}
}

func (m *Manager) navigateFilteredKeys(direction int, _ []TargetKey) {
	visibleRows := len(m.listWidget.Rows)
	if visibleRows == 0 {
		return
	}

	filteredIndices := m.listWidget.GetFilteredIndices()

	currentRow := m.listWidget.SelectedRow

	if direction > 0 {
		currentRow = (currentRow + 1) % visibleRows
	} else {
		currentRow = (currentRow - 1 + visibleRows) % visibleRows
	}

	m.listWidget.SelectedRow = currentRow

	if len(filteredIndices) > 0 {
		if currentRow >= len(filteredIndices) {
			if m.logBuffer != nil {
				m.logBuffer.AddLogEntry(LogLevelWarning, "NavigateFiltered",
					fmt.Sprintf("Row index out of bounds: currentRow=%d >= filteredIndices=%d",
						currentRow, len(filteredIndices)),
					TargetKey{})
			}
		} else if currentRow >= 0 {
			originalIdx := filteredIndices[currentRow]
			if originalIdx < len(m.itemToKeyIndex) {
				keyIdx := m.itemToKeyIndex[originalIdx]
				if keyIdx >= 0 {
					m.currentKeyIndex = keyIdx
				}
			} else {
				if m.logBuffer != nil {
					m.logBuffer.AddLogEntry(LogLevelWarning, "NavigateFiltered",
						fmt.Sprintf("Original index out of bounds: originalIdx=%d >= itemToKeyIndex=%d",
							originalIdx, len(m.itemToKeyIndex)),
						TargetKey{})
				}
			}
		}
	}
}

func (m *Manager) navigateAllKeys(direction int, _ []TargetKey) {
	visibleRows := len(m.listWidget.Rows)
	if visibleRows == 0 {
		return
	}

	currentPos := m.listWidget.SelectedRow
	if direction > 0 {
		currentPos = (currentPos + 1) % visibleRows
	} else {
		currentPos = (currentPos - 1 + visibleRows) % visibleRows
	}
	m.listWidget.SelectedRow = currentPos

	filteredIndices := m.listWidget.GetFilteredIndices()
	if currentPos < len(filteredIndices) {
		originalIdx := filteredIndices[currentPos]
		if originalIdx < len(m.itemToKeyIndex) {
			keyIdx := m.itemToKeyIndex[originalIdx]
			if keyIdx >= 0 {
				m.currentKeyIndex = keyIdx
			}
		}
	}
}
