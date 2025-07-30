package tui

import (
	"strings"

	"github.com/gizak/termui/v3/widgets"
)

const spaceKey = "<Space>"

type FilteredList struct {
	*widgets.List
	searchMode      bool
	searchQuery     string
	allRows         []string
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
		filteredIndices: []int{},
		filteredRows:    []string{},
	}
}

func (fl *FilteredList) SetRows(rows []string) {
	fl.allRows = rows
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
	} else if len(fl.searchQuery) < maxSearchLength && (len(char) == 1 || char == spaceKey) {
		if char == spaceKey {
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
		for i, row := range fl.allRows {
			if strings.Contains(strings.ToLower(row), query) {
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
