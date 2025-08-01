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
	collapsedGroups map[string]bool
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
		collapsedGroups: make(map[string]bool),
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
			if fl.rowMetadata[i].GroupID != "" && !fl.rowMetadata[i].IsHeader && fl.collapsedGroups[fl.rowMetadata[i].GroupID] {
				continue
			}
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
				if includeRow && !fl.rowMetadata[i].IsHeader && fl.collapsedGroups[fl.rowMetadata[i].GroupID] {
					includeRow = false
				}
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
	for displayIdx, originalIdx := range fl.filteredIndices {
		if originalIdx < len(fl.rowMetadata) && fl.rowMetadata[originalIdx].IsSelectable {
			selectableIndices = append(selectableIndices, displayIdx)
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

func (fl *FilteredList) ToggleGroupCollapse(groupID string) {
	if groupID == "" {
		return
	}
	fl.collapsedGroups[groupID] = !fl.collapsedGroups[groupID]
	fl.updateFiltered()
}

func (fl *FilteredList) IsGroupCollapsed(groupID string) bool {
	return fl.collapsedGroups[groupID]
}

func (fl *FilteredList) GetGroupAtIndex(index int) string {
	if index >= 0 && index < len(fl.filteredIndices) {
		originalIdx := fl.filteredIndices[index]
		if originalIdx < len(fl.rowMetadata) {
			return fl.rowMetadata[originalIdx].GroupID
		}
	}
	return ""
}

func (fl *FilteredList) IsHeaderAtIndex(index int) bool {
	if index >= 0 && index < len(fl.filteredIndices) {
		originalIdx := fl.filteredIndices[index]
		if originalIdx < len(fl.rowMetadata) {
			return fl.rowMetadata[originalIdx].IsHeader
		}
	}
	return false
}

func (fl *FilteredList) ToggleAllGroups() {
	groupsFound := make(map[string]bool)
	for _, meta := range fl.rowMetadata {
		if meta.GroupID != "" && meta.IsHeader {
			groupsFound[meta.GroupID] = true
		}
	}

	shouldExpand := false
	for groupID := range groupsFound {
		if fl.collapsedGroups[groupID] {
			shouldExpand = true
			break
		}
	}

	for groupID := range groupsFound {
		fl.collapsedGroups[groupID] = !shouldExpand
	}

	fl.updateFiltered()
}
