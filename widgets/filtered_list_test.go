package widgets

import (
	"reflect"
	"testing"
)

func TestFilteredList_UpdateSearch(t *testing.T) {
	tests := []struct {
		name         string
		setup        func() *FilteredList
		searchInputs []string
		wantQuery    string
		wantRows     []string
	}{
		{
			name: "no matches with query",
			setup: func() *FilteredList {
				fl := NewFilteredList()
				fl.SetRows([]string{"Apple", "Banana", "Cherry"})
				fl.searchMode = true
				return fl
			},
			searchInputs: []string{"x", "y", "z"},
			wantQuery:    "xyz",
			wantRows:     []string{"  No matches found"},
		},
		{
			name: "empty query shows all",
			setup: func() *FilteredList {
				fl := NewFilteredList()
				fl.SetRows([]string{"Apple", "Banana"})
				fl.searchMode = true
				fl.searchQuery = "test"
				fl.updateFiltered()
				return fl
			},
			searchInputs: []string{"<Backspace>", "<Backspace>", "<Backspace>", "<Backspace>"},
			wantQuery:    "",
			wantRows:     []string{"Apple", "Banana"},
		},
		{
			name: "max search length",
			setup: func() *FilteredList {
				fl := NewFilteredList()
				fl.SetRows([]string{"Item"})
				fl.searchMode = true
				fl.searchQuery = "aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeee"
				return fl
			},
			searchInputs: []string{"f", "g", "h"},
			wantQuery:    "aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeee",
			wantRows:     []string{"Item"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fl := tt.setup()

			for _, input := range tt.searchInputs {
				fl.UpdateSearch(input)
			}

			if fl.searchQuery != tt.wantQuery {
				t.Errorf("searchQuery = %q, want %q", fl.searchQuery, tt.wantQuery)
			}
			if !reflect.DeepEqual(fl.Rows, tt.wantRows) {
				t.Errorf("Rows = %v, want %v", fl.Rows, tt.wantRows)
			}
		})
	}
}

func TestFilteredList_GroupCollapseWithSearch(t *testing.T) {
	tests := []struct {
		name           string
		searchQuery    string
		collapseGroups []string
		wantVisible    []string
		wantIndices    int
	}{
		{
			name:           "collapsed group items hidden in search",
			searchQuery:    "Item",
			collapseGroups: []string{"Group B"},
			wantVisible: []string{
				"▼ Group A",
				"  Item A1",
				"  Item A2",
				"▼ Group B",
				"▼ Group C",
				"  Item C1",
			},
			wantIndices: 6,
		},
		{
			name:           "search shows headers of matching collapsed groups",
			searchQuery:    "B",
			collapseGroups: []string{"Group B"},
			wantVisible: []string{
				"▼ Group B",
			},
			wantIndices: 1,
		},
		{
			name:           "all groups collapsed with search",
			searchQuery:    "Group",
			collapseGroups: []string{"Group A", "Group B", "Group C"},
			wantVisible: []string{
				"▼ Group A",
				"▼ Group B",
				"▼ Group C",
			},
			wantIndices: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fl := NewFilteredList()
			rows := []string{
				"▼ Group A",
				"  Item A1",
				"  Item A2",
				"▼ Group B",
				"  Item B1",
				"  Item B2",
				"  Item B3",
				"▼ Group C",
				"  Item C1",
			}
			metadata := []RowMetadata{
				{GroupID: "Group A", IsHeader: true, IsSelectable: true},
				{GroupID: "Group A", IsHeader: false, IsSelectable: true},
				{GroupID: "Group A", IsHeader: false, IsSelectable: true},
				{GroupID: "Group B", IsHeader: true, IsSelectable: true},
				{GroupID: "Group B", IsHeader: false, IsSelectable: true},
				{GroupID: "Group B", IsHeader: false, IsSelectable: true},
				{GroupID: "Group B", IsHeader: false, IsSelectable: true},
				{GroupID: "Group C", IsHeader: true, IsSelectable: true},
				{GroupID: "Group C", IsHeader: false, IsSelectable: true},
			}
			fl.SetRowsWithMetadata(rows, metadata)

			for _, group := range tt.collapseGroups {
				fl.ToggleGroupCollapse(group)
			}

			fl.searchMode = true
			fl.searchQuery = tt.searchQuery
			fl.updateFiltered()

			if !reflect.DeepEqual(fl.Rows, tt.wantVisible) {
				t.Errorf("Visible rows = %v, want %v", fl.Rows, tt.wantVisible)
			}
			if len(fl.filteredIndices) != tt.wantIndices {
				t.Errorf("filteredIndices count = %d, want %d", len(fl.filteredIndices), tt.wantIndices)
			}
		})
	}
}

func TestFilteredList_NavigationEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		rows     []string
		metadata []RowMetadata
		testFunc func(*FilteredList) error
		wantErr  bool
	}{
		{
			name:     "empty list navigation",
			rows:     []string{},
			metadata: []RowMetadata{},
			testFunc: func(fl *FilteredList) error {
				if fl.IsHeaderAtIndex(0) {
					return nil
				}
				if fl.GetGroupAtIndex(0) != "" {
					return nil
				}
				return nil
			},
			wantErr: false,
		},
		{
			name:     "out of bounds index",
			rows:     []string{"Item 1", "Item 2"},
			metadata: []RowMetadata{{}, {}},
			testFunc: func(fl *FilteredList) error {
				if fl.IsHeaderAtIndex(5) {
					return nil
				}
				if fl.GetGroupAtIndex(-1) != "" {
					return nil
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "selectable indices with search",
			rows: []string{"Apple", "Banana", "Cherry"},
			metadata: []RowMetadata{
				{IsSelectable: true},
				{IsSelectable: false},
				{IsSelectable: true},
			},
			testFunc: func(fl *FilteredList) error {
				fl.searchMode = true
				fl.searchQuery = "a"
				fl.updateFiltered()

				selectable := fl.GetSelectableIndices()
				expected := []int{0}
				if !reflect.DeepEqual(selectable, expected) {
					t.Errorf("selectable = %v, want %v", selectable, expected)
				}
				return nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fl := NewFilteredList()
			fl.SetRowsWithMetadata(tt.rows, tt.metadata)

			err := tt.testFunc(fl)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFilteredList_SelectedRowBounds(t *testing.T) {
	fl := NewFilteredList()
	fl.SetRows([]string{"A", "B", "C"})

	fl.SelectedRow = 10
	fl.updateFiltered()
	if fl.SelectedRow != 0 {
		t.Errorf("SelectedRow after bounds correction = %d, want 0", fl.SelectedRow)
	}

	fl.searchMode = true
	fl.searchQuery = "xyz"
	fl.updateFiltered()
	if fl.SelectedRow != 0 {
		t.Errorf("SelectedRow with no matches = %d, want 0", fl.SelectedRow)
	}
}
