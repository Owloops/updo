package widgets

import (
	"strings"

	"github.com/gizak/termui/v3/widgets"
)

type RowMetadata struct {
	GroupID      string
	IsHeader     bool
	IsSelectable bool
}

type FilteredList struct {
	*widgets.List
	searchMode      bool
	searchQuery     string
	allRows         []string
	rowMetadata     []RowMetadata
	filteredIndices []int
	filteredRows    []string
	OnSearchChange  func(query string, filteredIndices []int)
}

func NewFilteredList() *FilteredList {
	list := widgets.NewList()
	return &FilteredList{
		List:            list,
		searchMode:      false,
		searchQuery:     "",
		allRows:         []string{},
		rowMetadata:     []RowMetadata{},
		filteredIndices: []int{},
		filteredRows:    []string{},
	}
}

func (fl *FilteredList) SetRows(rows []string) {
	fl.allRows = rows
	fl.rowMetadata = make([]RowMetadata, len(rows))
	for i := range fl.rowMetadata {
		fl.rowMetadata[i] = RowMetadata{
			GroupID:      "",
			IsHeader:     false,
			IsSelectable: true,
		}
	}
	fl.updateFiltered()
}

func (fl *FilteredList) SetRowsWithMetadata(rows []string, metadata []RowMetadata) {
	fl.allRows = rows
	fl.rowMetadata = metadata
	fl.updateFiltered()
}

func (fl *FilteredList) ToggleSearch() {
	fl.searchMode = !fl.searchMode
	if !fl.searchMode {
		fl.searchQuery = ""
		fl.updateFiltered()
	}
}

func (fl *FilteredList) UpdateSearch(char string) {
	if !fl.searchMode {
		return
	}

	const maxSearchLength = 50
	prevQuery := fl.searchQuery

	if char == "<Backspace>" || char == "<C-8>" {
		if len(fl.searchQuery) > 0 {
			fl.searchQuery = fl.searchQuery[:len(fl.searchQuery)-1]
		}
	} else if len(fl.searchQuery) < maxSearchLength && (len(char) == 1 || char == "<Space>") {
		if char == "<Space>" {
			fl.searchQuery += " "
		} else {
			fl.searchQuery += char
		}
	}

	if prevQuery != fl.searchQuery {
		fl.updateFiltered()
		if fl.SelectedRow >= len(fl.filteredRows) {
			fl.SelectedRow = 0
		}
	}
}

func (fl *FilteredList) IsSearchMode() bool {
	return fl.searchMode
}

func (fl *FilteredList) GetQuery() string {
	return fl.searchQuery
}

func (fl *FilteredList) GetFilteredIndices() []int {
	return fl.filteredIndices
}

func (fl *FilteredList) updateFiltered() {
	fl.filteredIndices = make([]int, 0, len(fl.allRows))
	fl.filteredRows = make([]string, 0, len(fl.allRows))

	if fl.searchQuery == "" {
		for i, row := range fl.allRows {
			fl.filteredIndices = append(fl.filteredIndices, i)
			fl.filteredRows = append(fl.filteredRows, row)
		}
	} else {
		query := strings.ToLower(fl.searchQuery)

		matchingGroups := make(map[string]bool)
		for i, row := range fl.allRows {
			if strings.Contains(strings.ToLower(row), query) {
				if fl.rowMetadata[i].GroupID != "" {
					matchingGroups[fl.rowMetadata[i].GroupID] = true
				}
			}
		}

		for i, row := range fl.allRows {
			includeRow := false

			if fl.rowMetadata[i].GroupID != "" {
				includeRow = matchingGroups[fl.rowMetadata[i].GroupID]
			} else {
				includeRow = strings.Contains(strings.ToLower(row), query)
			}

			if includeRow {
				fl.filteredIndices = append(fl.filteredIndices, i)
				fl.filteredRows = append(fl.filteredRows, row)
			}
		}
	}

	if fl.searchMode && len(fl.filteredRows) == 0 && fl.searchQuery != "" {
		fl.Rows = []string{"  No matches found"}
		fl.filteredIndices = []int{}
	} else {
		fl.Rows = fl.filteredRows
	}

	if fl.SelectedRow >= len(fl.Rows) {
		fl.SelectedRow = 0
	}

	if fl.OnSearchChange != nil {
		fl.OnSearchChange(fl.searchQuery, fl.filteredIndices)
	}
}

func (fl *FilteredList) GetSelectableIndices() []int {
	selectableIndices := make([]int, 0)
	for _, idx := range fl.filteredIndices {
		if idx < len(fl.rowMetadata) && fl.rowMetadata[idx].IsSelectable {
			selectableIndices = append(selectableIndices, idx)
		}
	}
	return selectableIndices
}

func (fl *FilteredList) GetFilteredDisplayIndices() map[int]int {
	displayMap := make(map[int]int)
	for displayIdx, originalIdx := range fl.filteredIndices {
		displayMap[originalIdx] = displayIdx
	}
	return displayMap
}
