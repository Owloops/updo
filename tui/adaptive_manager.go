package tui

import (
	"fmt"
	"strings"

	"github.com/Owloops/updo/config"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

const (
	searchText   = "Press / to search"
	searchTitle  = " Search "
	targetsTitle = " Targets "
	statusWait   = "WAIT"
	statusDown   = "DOWN"
)

type NodeValue string

func (nv NodeValue) String() string { return string(nv) }

type DisplayMode int

const (
	SinglePane DisplayMode = iota
	FlatList
	TreeView
)

type AdaptiveManager struct {
	searchWidget *widgets.Paragraph
	treeWidget   *widgets.Tree
	listWidget   *FilteredList

	displayMode           DisplayMode
	isSearching           bool
	targets               []config.Target
	dataStore             *DataStore
	targetNodesByName     map[string]*widgets.TreeNode
	currentTargetIndex    int
	totalVisibleTreeNodes int

	reusableListItems   []string
	reusableSearchItems []string
}

func NewAdaptiveManager(dataStore *DataStore) *AdaptiveManager {
	return &AdaptiveManager{
		dataStore:         dataStore,
		targetNodesByName: make(map[string]*widgets.TreeNode),
	}
}

func (am *AdaptiveManager) Initialize(targets []config.Target, hasRegions bool) {
	am.targets = targets
	am.determineDisplayMode(targets, hasRegions)
	am.initializeWidgets()
	am.populateContent()
}

func (am *AdaptiveManager) determineDisplayMode(targets []config.Target, hasRegions bool) {
	switch {
	case len(targets) == 1 && !hasRegions:
		am.displayMode = SinglePane
	case !hasRegions:
		am.displayMode = FlatList
	default:
		am.displayMode = TreeView
	}
}

func (am *AdaptiveManager) initializeWidgets() {
	am.createSearchWidget()

	switch am.displayMode {
	case FlatList:
		am.createFlatListWidget()
	case TreeView:
		am.createTreeWidget()
		am.createSearchListWidget()
	}
}

func (am *AdaptiveManager) createSearchWidget() {
	am.searchWidget = widgets.NewParagraph()
	am.searchWidget.Border = true
	am.searchWidget.BorderStyle.Fg = ui.ColorCyan
	am.searchWidget.TitleStyle.Fg = ui.ColorCyan
	am.searchWidget.TitleStyle.Modifier = ui.ModifierBold
	am.searchWidget.TextStyle.Fg = ui.ColorWhite

	if am.displayMode == SinglePane {
		am.searchWidget.Title = " Status "
		am.searchWidget.Text = "Initializing..."
	} else {
		am.searchWidget.Title = searchTitle
		am.searchWidget.Text = searchText
	}
}

func (am *AdaptiveManager) createFlatListWidget() {
	am.listWidget = NewFilteredList()
	am.listWidget.Title = targetsTitle
	am.listWidget.BorderStyle.Fg = ui.ColorCyan
	am.listWidget.TitleStyle.Fg = ui.ColorCyan
	am.listWidget.TitleStyle.Modifier = ui.ModifierBold
	am.listWidget.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorCyan, ui.ModifierBold)
	am.setupSearchCallbacks(false)
}

func (am *AdaptiveManager) createTreeWidget() {
	am.treeWidget = widgets.NewTree()
	am.treeWidget.Title = " Targets "
	am.treeWidget.BorderStyle.Fg = ui.ColorCyan
	am.treeWidget.TitleStyle.Fg = ui.ColorCyan
	am.treeWidget.TitleStyle.Modifier = ui.ModifierBold
	am.treeWidget.TextStyle = ui.NewStyle(ui.ColorWhite)
	am.treeWidget.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorCyan, ui.ModifierBold)
	am.treeWidget.WrapText = false
}

func (am *AdaptiveManager) createSearchListWidget() {
	am.listWidget = NewFilteredList()
	am.listWidget.Title = " Search Results "
	am.listWidget.BorderStyle.Fg = ui.ColorMagenta
	am.listWidget.TitleStyle.Fg = ui.ColorMagenta
	am.listWidget.TitleStyle.Modifier = ui.ModifierBold
	am.listWidget.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorMagenta, ui.ModifierBold)
	am.setupSearchCallbacks(true)
}

func (am *AdaptiveManager) setupSearchCallbacks(isTreeMode bool) {
	am.listWidget.OnSearchChange = func(query string, filteredIndices []int) {
		am.handleSearchChange(query, filteredIndices, isTreeMode)
	}
}

func (am *AdaptiveManager) handleSearchChange(query string, filteredIndices []int, isTreeMode bool) {
	if !am.listWidget.IsSearchMode() {
		am.resetSearchUI(isTreeMode)
		return
	}

	if query != "" {
		am.updateActiveSearchUI(query, len(filteredIndices))
		am.handleSearchResults(filteredIndices, isTreeMode)
	} else {
		am.updateEmptySearchUI()
	}

	if !isTreeMode {
		am.buildFlatList()
	}
}

func (am *AdaptiveManager) resetSearchUI(isTreeMode bool) {
	am.searchWidget.Text = searchText
	am.searchWidget.TextStyle.Fg = ui.ColorWhite
	am.searchWidget.BorderStyle.Fg = ui.ColorCyan
	am.searchWidget.TitleStyle.Fg = ui.ColorCyan

	if isTreeMode {
		am.isSearching = false
	} else {
		am.listWidget.Title = targetsTitle
		am.listWidget.BorderStyle.Fg = ui.ColorCyan
		am.listWidget.TitleStyle.Fg = ui.ColorCyan
		am.listWidget.SelectedRow = am.currentTargetIndex
	}
}

func (am *AdaptiveManager) updateActiveSearchUI(query string, resultCount int) {
	am.searchWidget.Text = fmt.Sprintf("→ %s", query)
	am.searchWidget.TextStyle.Fg = ui.ColorGreen
	am.searchWidget.BorderStyle.Fg = ui.ColorGreen
	am.searchWidget.TitleStyle.Fg = ui.ColorGreen
	am.listWidget.Title = fmt.Sprintf(" Found %d ", resultCount)
	am.listWidget.BorderStyle.Fg = ui.ColorGreen
	am.listWidget.TitleStyle.Fg = ui.ColorGreen
}

func (am *AdaptiveManager) updateEmptySearchUI() {
	am.searchWidget.Text = "Type to search..."
	am.searchWidget.TextStyle.Fg = ui.ColorYellow
	am.searchWidget.BorderStyle.Fg = ui.ColorYellow
	am.searchWidget.TitleStyle.Fg = ui.ColorYellow
	am.listWidget.Title = " Search "
	am.listWidget.BorderStyle.Fg = ui.ColorYellow
	am.listWidget.TitleStyle.Fg = ui.ColorYellow
	am.listWidget.SelectedRow = 0
}

func (am *AdaptiveManager) handleSearchResults(filteredIndices []int, isTreeMode bool) {
	if isTreeMode {
		am.isSearching = true
		if len(filteredIndices) > 0 {
			am.listWidget.SelectedRow = 0
		}
	} else {
		am.maintainFlatListSelection(filteredIndices)
	}
}

func (am *AdaptiveManager) maintainFlatListSelection(filteredIndices []int) {
	targetVisible := false
	if len(filteredIndices) > 0 {
		for i, idx := range filteredIndices {
			if idx == am.currentTargetIndex {
				am.listWidget.SelectedRow = i
				targetVisible = true
				break
			}
		}
	}

	if !targetVisible && len(filteredIndices) > 0 {
		am.listWidget.SelectedRow = 0
		am.currentTargetIndex = filteredIndices[0]
	}
}

func (am *AdaptiveManager) populateContent() {
	switch am.displayMode {
	case FlatList:
		am.buildFlatList()
	case TreeView:
		am.buildTree()
	}
}

func (am *AdaptiveManager) buildFlatList() {
	if am.listWidget == nil {
		return
	}

	if cap(am.reusableListItems) < len(am.targets) {
		am.reusableListItems = make([]string, len(am.targets))
	} else {
		am.reusableListItems = am.reusableListItems[:len(am.targets)]
	}

	for i, target := range am.targets {
		displayName := am.getDisplayName(target)
		if len(displayName) > 35 {
			displayName = displayName[:32] + "..."
		}

		status := am.getTargetStatus(target)
		prefix := "  "
		if i == am.currentTargetIndex {
			prefix = "▶ "
		}

		am.reusableListItems[i] = fmt.Sprintf("%s[%s] %s", prefix, status, displayName)
	}

	am.listWidget.SetRows(am.reusableListItems)
}

func (am *AdaptiveManager) getTargetStatus(target config.Target) string {
	localKey := NewLocalTargetKey(target)
	if data, exists := am.dataStore.GetTargetData(localKey); exists {
		return am.formatStatus(data)
	}

	for _, key := range am.dataStore.GetAllTargetKeys() {
		if key.TargetName == target.Name && key.IsRegional() {
			if data, exists := am.dataStore.GetTargetData(key); exists {
				return am.formatStatus(data)
			}
		}
	}

	return statusWait
}

func (am *AdaptiveManager) formatStatus(data TargetData) string {
	if data.Result.IsUp {
		return "UP"
	}
	return statusDown
}

func (am *AdaptiveManager) buildTree() {
	if am.treeWidget == nil {
		return
	}

	var treeNodes []*widgets.TreeNode
	am.targetNodesByName = make(map[string]*widgets.TreeNode)
	am.totalVisibleTreeNodes = 0

	for _, target := range am.targets {
		displayName := am.getDisplayName(target)

		targetNode := &widgets.TreeNode{
			Value: NodeValue(displayName),
			Nodes: []*widgets.TreeNode{},
		}

		am.targetNodesByName[target.Name] = targetNode
		treeNodes = append(treeNodes, targetNode)
		am.totalVisibleTreeNodes++
	}

	am.treeWidget.SetNodes(treeNodes)
}

func (am *AdaptiveManager) getDisplayName(target config.Target) string {
	displayName := target.Name
	if displayName == "" || strings.HasPrefix(displayName, "Target-") {
		displayName = target.URL
		displayName = strings.TrimPrefix(displayName, "https://")
		displayName = strings.TrimPrefix(displayName, "http://")
	}
	return displayName
}

func (am *AdaptiveManager) AddRegion(target config.Target, region string) {
	if am.displayMode != TreeView {
		return
	}

	targetNode, exists := am.targetNodesByName[target.Name]
	if !exists {
		return
	}

	for _, regionNode := range targetNode.Nodes {
		if regionValue, ok := regionNode.Value.(NodeValue); ok {
			if strings.HasPrefix(string(regionValue), region) {
				return
			}
		}
	}

	regionNode := &widgets.TreeNode{
		Value: NodeValue(fmt.Sprintf("%s [WAIT]", region)),
		Nodes: []*widgets.TreeNode{},
	}

	targetNode.Nodes = append(targetNode.Nodes, regionNode)
	am.totalVisibleTreeNodes++
}

func (am *AdaptiveManager) HandleUpdateEvent(event UpdateEvent) {
	if event.Type == TargetDataUpdateEvent {
		am.updateDisplayForTargetData(event.Key, event.Data)
	}
}

func (am *AdaptiveManager) updateDisplayForTargetData(key TargetKey, data TargetData) {
	switch am.displayMode {
	case SinglePane:
		am.updateSinglePaneStatus(data)
	case FlatList:
		am.buildFlatList()
	case TreeView:
		am.updateTreeStatus(key, data)
		if am.isSearching {
			am.updateSearchContent()
		}
	}
}

func (am *AdaptiveManager) UpdateTargetStatus(key TargetKey, data TargetData) {
	am.updateDisplayForTargetData(key, data)
}

func (am *AdaptiveManager) updateSinglePaneStatus(data TargetData) {
	if am.searchWidget == nil {
		return
	}

	status := statusDown
	if data.Result.IsUp {
		responseTime := data.Result.ResponseTime.Milliseconds()
		status = fmt.Sprintf("UP %dms", responseTime)
	}

	am.searchWidget.Text = fmt.Sprintf("Status: %s | Target: %s", status, am.getDisplayName(data.Target))
}

func (am *AdaptiveManager) updateTreeStatus(key TargetKey, data TargetData) {
	if key.IsLocal() {
		return
	}

	targetNode, exists := am.targetNodesByName[key.TargetName]
	if !exists {
		return
	}

	for _, regionNode := range targetNode.Nodes {
		if regionValue, ok := regionNode.Value.(NodeValue); ok {
			if strings.HasPrefix(string(regionValue), key.Region) {
				status := statusDown
				if data.Result.IsUp {
					responseTime := data.Result.ResponseTime.Milliseconds()
					status = fmt.Sprintf("UP %3dms", responseTime)
				}
				regionNode.Value = NodeValue(fmt.Sprintf("%s [%s]", key.Region, status))
				break
			}
		}
	}
}

func (am *AdaptiveManager) GetCurrentSelection() (config.Target, string, TargetKey) {
	switch am.displayMode {
	case SinglePane:
		if len(am.targets) > 0 {
			return am.targets[0], "", NewLocalTargetKey(am.targets[0])
		}
	case FlatList:
		return am.getFlatListSelection()
	case TreeView:
		return am.getTreeViewSelection()
	}
	return config.Target{}, "", TargetKey{}
}

func (am *AdaptiveManager) getFlatListSelection() (config.Target, string, TargetKey) {
	if am.listWidget != nil && am.listWidget.IsSearchMode() {
		filteredIndices := am.listWidget.GetFilteredIndices()
		selectedRow := am.listWidget.SelectedRow
		if selectedRow >= 0 && selectedRow < len(filteredIndices) {
			originalIndex := filteredIndices[selectedRow]
			if originalIndex >= 0 && originalIndex < len(am.targets) {
				target := am.targets[originalIndex]
				return target, "", NewLocalTargetKey(target)
			}
		}
		return config.Target{}, "", TargetKey{}
	}

	if am.currentTargetIndex >= 0 && am.currentTargetIndex < len(am.targets) {
		target := am.targets[am.currentTargetIndex]
		return target, "", NewLocalTargetKey(target)
	}
	return config.Target{}, "", TargetKey{}
}

func (am *AdaptiveManager) getTreeViewSelection() (config.Target, string, TargetKey) {
	if am.isSearching && am.listWidget != nil && am.listWidget.IsSearchMode() {
		return am.getTreeSearchSelection()
	}

	if am.treeWidget != nil {
		return am.getTreeNormalSelection()
	}

	if len(am.targets) > 0 {
		return am.targets[0], "", NewLocalTargetKey(am.targets[0])
	}
	return config.Target{}, "", TargetKey{}
}

func (am *AdaptiveManager) getTreeSearchSelection() (config.Target, string, TargetKey) {
	selectedRow := am.listWidget.SelectedRow
	if selectedRow >= 0 && selectedRow < len(am.listWidget.Rows) {
		selectedItem := am.listWidget.Rows[selectedRow]
		if selectedItem == "  No matches found" {
			return config.Target{}, "", TargetKey{}
		}
		return am.parseSearchItem(selectedItem)
	}
	return config.Target{}, "", TargetKey{}
}

func (am *AdaptiveManager) parseSearchItem(selectedItem string) (config.Target, string, TargetKey) {
	if strings.Contains(selectedItem, " → ") {
		parts := strings.Split(selectedItem, " → ")
		if len(parts) >= 2 {
			targetDisplay := strings.TrimSpace(parts[0])
			regionPart := strings.TrimSpace(parts[1])

			if strings.Contains(regionPart, " [") {
				regionParts := strings.Split(regionPart, " [")
				region := strings.TrimSpace(regionParts[0])

				for _, target := range am.targets {
					if am.getDisplayName(target) == targetDisplay {
						return target, region, NewRegionalTargetKey(target, region)
					}
				}
			}
		}
		return config.Target{}, "", TargetKey{}
	}

	for _, target := range am.targets {
		if am.getDisplayName(target) == selectedItem {
			return target, "", NewLocalTargetKey(target)
		}
	}
	return config.Target{}, "", TargetKey{}
}

func (am *AdaptiveManager) getTreeNormalSelection() (config.Target, string, TargetKey) {
	selectedNode := am.treeWidget.SelectedNode()
	if selectedNode == nil {
		return config.Target{}, "", TargetKey{}
	}

	nodeValue, ok := selectedNode.Value.(NodeValue)
	if !ok {
		return config.Target{}, "", TargetKey{}
	}

	nodeStr := string(nodeValue)

	if am.isRegionNode(nodeStr) {
		return am.getRegionSelection(selectedNode, nodeStr)
	}
	return am.getTargetSelection(nodeStr)
}

func (am *AdaptiveManager) getRegionSelection(selectedNode *widgets.TreeNode, nodeStr string) (config.Target, string, TargetKey) {
	parts := strings.Split(nodeStr, " [")
	if len(parts) == 0 {
		return config.Target{}, "", TargetKey{}
	}

	region := strings.TrimSpace(parts[0])
	for targetName, targetNode := range am.targetNodesByName {
		for _, regionNode := range targetNode.Nodes {
			if regionNode == selectedNode {
				for _, target := range am.targets {
					if target.Name == targetName {
						return target, region, NewRegionalTargetKey(target, region)
					}
				}
			}
		}
	}
	return config.Target{}, "", TargetKey{}
}

func (am *AdaptiveManager) getTargetSelection(nodeStr string) (config.Target, string, TargetKey) {
	for _, target := range am.targets {
		if am.getDisplayName(target) == nodeStr {
			if targetNode, exists := am.targetNodesByName[target.Name]; exists && len(targetNode.Nodes) > 0 {
				if firstRegion, ok := targetNode.Nodes[0].Value.(NodeValue); ok {
					regionStr := string(firstRegion)
					if strings.Contains(regionStr, "[") {
						parts := strings.Split(regionStr, " [")
						if len(parts) >= 1 {
							region := strings.TrimSpace(parts[0])
							return target, region, NewRegionalTargetKey(target, region)
						}
					}
				}
			}
			return target, "", NewLocalTargetKey(target)
		}
	}
	return config.Target{}, "", TargetKey{}
}

func (am *AdaptiveManager) Navigate(direction int) {
	switch am.displayMode {
	case FlatList:
		am.navigateFlatList(direction)
	case TreeView:
		am.navigateTreeView(direction)
	}
}

func (am *AdaptiveManager) navigateFlatList(direction int) {
	if am.listWidget != nil && am.listWidget.IsSearchMode() {
		am.navigateFilteredList(direction)
	} else {
		am.navigateNormalList(direction)
	}
	am.buildFlatList()
}

func (am *AdaptiveManager) navigateFilteredList(direction int) {
	filteredIndices := am.listWidget.GetFilteredIndices()
	if len(filteredIndices) == 0 {
		return
	}

	currentFilteredIndex := -1
	for i, idx := range filteredIndices {
		if idx == am.currentTargetIndex {
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

	am.currentTargetIndex = filteredIndices[currentFilteredIndex]
	am.listWidget.SelectedRow = currentFilteredIndex
}

func (am *AdaptiveManager) navigateNormalList(direction int) {
	if len(am.targets) > 0 {
		if direction > 0 {
			am.currentTargetIndex = (am.currentTargetIndex + 1) % len(am.targets)
		} else {
			am.currentTargetIndex = (am.currentTargetIndex - 1 + len(am.targets)) % len(am.targets)
		}
	}
}

func (am *AdaptiveManager) navigateTreeView(direction int) {
	if am.isSearching && am.listWidget != nil {
		rows := len(am.listWidget.Rows)
		if rows > 0 {
			if direction > 0 {
				am.listWidget.SelectedRow = (am.listWidget.SelectedRow + 1) % rows
			} else {
				am.listWidget.SelectedRow = (am.listWidget.SelectedRow - 1 + rows) % rows
			}
		}
	} else if am.treeWidget != nil {
		totalNodes := am.totalVisibleTreeNodes
		if totalNodes > 0 {
			if direction > 0 {
				am.treeWidget.SelectedRow = (am.treeWidget.SelectedRow + 1) % totalNodes
			} else {
				am.treeWidget.SelectedRow = (am.treeWidget.SelectedRow - 1 + totalNodes) % totalNodes
			}
		}
	}
}

func (am *AdaptiveManager) ToggleExpansion() bool {
	if am.displayMode == TreeView && am.treeWidget != nil {
		am.treeWidget.ToggleExpand()
		return true
	}
	return false
}

func (am *AdaptiveManager) ToggleSearch() {
	if am.listWidget != nil && (am.displayMode == FlatList || am.displayMode == TreeView) {
		am.listWidget.ToggleSearch()
		if am.displayMode == TreeView {
			am.updateSearchContent()
		}
	}
}

func (am *AdaptiveManager) UpdateSearch(input string) {
	if am.listWidget != nil && am.listWidget.IsSearchMode() && (am.displayMode == FlatList || am.displayMode == TreeView) {
		am.listWidget.UpdateSearch(input)
	}
}

func (am *AdaptiveManager) IsSearchMode() bool {
	return am.isSearching || (am.listWidget != nil && am.listWidget.IsSearchMode())
}

func (am *AdaptiveManager) GetDisplayMode() DisplayMode {
	return am.displayMode
}

func (am *AdaptiveManager) GetActiveWidget() any {
	switch am.displayMode {
	case SinglePane:
		return nil
	case FlatList:
		return am.listWidget
	case TreeView:
		if am.isSearching && am.listWidget != nil {
			return am.listWidget
		}
		return am.treeWidget
	}
	return nil
}

func (am *AdaptiveManager) GetSearchWidget() *widgets.Paragraph {
	return am.searchWidget
}

func (am *AdaptiveManager) GetListWidgetForSearch() *FilteredList {
	return am.listWidget
}

func (am *AdaptiveManager) updateSearchContent() {
	if am.displayMode != TreeView || am.listWidget == nil {
		return
	}

	am.reusableSearchItems = am.reusableSearchItems[:0]
	for _, target := range am.targets {
		targetNode := am.targetNodesByName[target.Name]
		if targetNode == nil {
			continue
		}

		targetValue, ok := targetNode.Value.(NodeValue)
		if !ok {
			continue
		}

		targetDisplay := string(targetValue)

		if len(targetNode.Nodes) > 0 {
			for _, regionNode := range targetNode.Nodes {
				if regionValue, ok := regionNode.Value.(NodeValue); ok {
					regionStr := string(regionValue)
					if strings.Contains(regionStr, "[") {
						parts := strings.Split(regionStr, " [")
						if len(parts) >= 2 {
							region := strings.TrimSpace(parts[0])
							status := strings.TrimSuffix(parts[1], "]")
							searchItem := fmt.Sprintf("%s → %s [%s]", targetDisplay, region, status)
							am.reusableSearchItems = append(am.reusableSearchItems, searchItem)
						}
					} else {
						searchItem := fmt.Sprintf("%s → %s", targetDisplay, regionStr)
						am.reusableSearchItems = append(am.reusableSearchItems, searchItem)
					}
				}
			}
		} else {
			am.reusableSearchItems = append(am.reusableSearchItems, targetDisplay)
		}
	}
	am.listWidget.SetRows(am.reusableSearchItems)
}

func (am *AdaptiveManager) isRegionNode(nodeText string) bool {
	return strings.Contains(nodeText, "[")
}
