package widgets

import (
	"image"
	"strings"

	ui "github.com/Owloops/termui/v3"
	"github.com/Owloops/termui/v3/widgets"
	rw "github.com/mattn/go-runewidth"
)

const (
	_maxSearchLength = 50
	_backspaceKey    = "<Backspace>"
	_ctrlBackspace   = "<C-8>"
	_spaceKey        = "<Space>"
	_selectionMarker = '▌'
	_childIndent     = 2
	_rightWidth      = 6
	_minSparkWidth   = 3
)

type RowMetadata struct {
	GroupID      string
	IsHeader     bool
	IsSelectable bool
	Label        string
	Right        string
	RightStyle   ui.Style
	Spark        []rune
}

// ListLook styles the rows of a framed FilteredList.
type ListLook struct {
	Text     ui.Style
	Header   ui.Style
	Chevron  ui.Style
	Marker   ui.Style
	Spark    ui.Style
	Selected ui.Style
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

	Frame  *Frame
	Look   ListLook
	topRow int
}

func NewFilteredList() *FilteredList {
	return &FilteredList{
		List:            widgets.NewList(),
		collapsedGroups: make(map[string]bool),
	}
}

// Draw renders the stock termui list, or the framed look when Frame is set.
func (fl *FilteredList) Draw(buf *ui.Buffer) {
	if fl.Frame == nil {
		fl.List.Draw(buf)
		return
	}

	rect := fl.GetRect()
	fl.Frame.Draw(buf, rect)
	inner := fl.Frame.Inner(rect)
	if inner.Empty() {
		return
	}

	if fl.SelectedRow >= fl.topRow+inner.Dy() {
		fl.topRow = fl.SelectedRow - inner.Dy() + 1
	} else if fl.SelectedRow < fl.topRow {
		fl.topRow = fl.SelectedRow
	}
	fl.topRow = max(min(fl.topRow, len(fl.Rows)-1), 0)

	for row := fl.topRow; row < len(fl.Rows) && row-fl.topRow < inner.Dy(); row++ {
		fl.drawRow(buf, row, inner, inner.Min.Y+row-fl.topRow)
	}
}

func (fl *FilteredList) drawRow(buf *ui.Buffer, row int, inner image.Rectangle, y int) {
	var meta RowMetadata
	if row < len(fl.filteredIndices) && fl.filteredIndices[row] < len(fl.rowMetadata) {
		meta = fl.rowMetadata[fl.filteredIndices[row]]
	}

	isSelected := row == fl.SelectedRow && len(fl.filteredIndices) > 0
	restyle := func(style ui.Style) ui.Style {
		if isSelected {
			style.Bg = fl.Look.Selected.Bg
			style.Modifier |= fl.Look.Selected.Modifier
			if style.Fg == fl.Look.Text.Fg || style.Fg == fl.Look.Header.Fg {
				style.Fg = fl.Look.Selected.Fg
			}
		}
		return style
	}

	if isSelected {
		buf.Fill(ui.NewCell(' ', restyle(fl.Look.Text)), image.Rect(inner.Min.X, y, inner.Max.X, y+1))
		buf.SetCell(ui.NewCell(_selectionMarker, restyle(fl.Look.Marker)), image.Pt(inner.Min.X, y))
	}

	base := fl.Look.Text
	text := fl.Rows[row]
	if meta.IsHeader {
		base = fl.Look.Header
	} else if trimmed := strings.TrimLeft(text, " "); trimmed != text {
		text = strings.Repeat(" ", _childIndent) + trimmed
	}

	cells := ui.ParseStyles(text, base)
	x := inner.Min.X + _childIndent

	rightStart := inner.Max.X - 1
	if meta.Right != "" {
		rightStart -= max(rw.StringWidth(meta.Right), _rightWidth)
		DrawTextRight(buf, meta.Right, restyle(meta.RightStyle), rightStart, y, inner.Max.X-1)
		rightStart--
	}
	if width := min(len(meta.Spark), rightStart-x-len(cells)-1); width >= _minSparkWidth {
		style := fl.Look.Spark
		if isSelected {
			style = fl.Look.Marker
		}
		DrawText(buf, string(meta.Spark[len(meta.Spark)-width:]), restyle(style), rightStart-width, y, rightStart)
		rightStart -= width + 1
	}

	for i, cell := range cells {
		if x >= rightStart {
			buf.SetCell(ui.NewCell(ui.ELLIPSES, restyle(base)), image.Pt(x-1, y))
			break
		}
		style := cell.Style
		if meta.IsHeader && i == 0 {
			style = fl.Look.Chevron
		}
		x = DrawText(buf, string(cell.Rune), restyle(style), x, y, rightStart)
	}
}

func (fl *FilteredList) SetRows(rows []string) {
	fl.allRows = rows
	fl.rowMetadata = make([]RowMetadata, len(rows))
	for i := range fl.rowMetadata {
		fl.rowMetadata[i] = RowMetadata{
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
		var selectedOriginalIdx int
		if fl.SelectedRow >= 0 && fl.SelectedRow < len(fl.filteredIndices) {
			selectedOriginalIdx = fl.filteredIndices[fl.SelectedRow]
		}

		fl.searchQuery = ""
		fl.updateFiltered()

		for i, idx := range fl.filteredIndices {
			if idx == selectedOriginalIdx {
				fl.SelectedRow = i
				break
			}
		}
	}
}

func (fl *FilteredList) UpdateSearch(char string) {
	if !fl.searchMode {
		return
	}

	prevQuery := fl.searchQuery

	if char == _backspaceKey || char == _ctrlBackspace {
		if len(fl.searchQuery) > 0 {
			fl.searchQuery = fl.searchQuery[:len(fl.searchQuery)-1]
		}
	} else if len(fl.searchQuery) < _maxSearchLength && (len(char) == 1 || char == _spaceKey) {
		var newChar string
		if char == _spaceKey {
			newChar = " "
		} else {
			newChar = char
		}

		var builder strings.Builder
		builder.Grow(len(fl.searchQuery) + len(newChar))
		builder.WriteString(fl.searchQuery)
		builder.WriteString(newChar)
		fl.searchQuery = builder.String()
	}

	if prevQuery != fl.searchQuery {
		var selectedOriginalIdx = -1
		if fl.SelectedRow >= 0 && fl.SelectedRow < len(fl.filteredIndices) {
			selectedOriginalIdx = fl.filteredIndices[fl.SelectedRow]
		}

		fl.updateFiltered()

		if selectedOriginalIdx >= 0 {
			newSelectedRow := -1
			for i, idx := range fl.filteredIndices {
				if idx == selectedOriginalIdx {
					newSelectedRow = i
					break
				}
			}
			if newSelectedRow >= 0 {
				fl.SelectedRow = newSelectedRow
			} else if fl.SelectedRow >= len(fl.filteredRows) {
				fl.SelectedRow = 0
			}
		} else if fl.SelectedRow >= len(fl.filteredRows) {
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
		for i := range fl.allRows {
			if fl.matches(i, query) {
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
				includeRow = fl.matches(i, query)
			}

			if includeRow {
				fl.filteredIndices = append(fl.filteredIndices, i)
				fl.filteredRows = append(fl.filteredRows, row)
			}
		}
	}

	if fl.searchMode && len(fl.filteredRows) == 0 && fl.searchQuery != "" {
		fl.Rows = []string{"  No matches found"}
		fl.filteredIndices = nil
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

func (fl *FilteredList) matches(index int, query string) bool {
	text := fl.allRows[index]
	if label := fl.rowMetadata[index].Label; label != "" {
		text = label
	}
	return strings.Contains(strings.ToLower(text), query)
}

func (fl *FilteredList) GetSelectableIndices() []int {
	selectableIndices := make([]int, 0, len(fl.filteredIndices))
	for displayIdx, originalIdx := range fl.filteredIndices {
		if originalIdx < len(fl.rowMetadata) && fl.rowMetadata[originalIdx].IsSelectable {
			selectableIndices = append(selectableIndices, displayIdx)
		}
	}
	return selectableIndices
}

func (fl *FilteredList) GetFilteredDisplayIndices() map[int]int {
	displayMap := make(map[int]int, len(fl.filteredIndices))
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
