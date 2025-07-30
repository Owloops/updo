package tui

import (
	"strings"

	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
	ui "github.com/gizak/termui/v3"
)

type PlotHistory struct {
	UptimeData       []float64
	ResponseTimeData []float64
}

type RefactoredManager struct {
	targets    []config.Target
	dataStore  *DataStore
	dispatcher *EventDispatcher
	hasRegions bool
	isSingle   bool

	adaptiveManager *AdaptiveManager
	detailsManager  *DetailsPanelManager
	grid            *ui.Grid
	termWidth       int
	termHeight      int
}

func NewRefactoredManager(targets []config.Target, regions []string) *RefactoredManager {
	willHaveRegions := len(regions) > 0
	dataStore := NewDataStore()
	m := &RefactoredManager{
		targets:         targets,
		dataStore:       dataStore,
		dispatcher:      NewEventDispatcher(),
		hasRegions:      willHaveRegions,
		isSingle:        len(targets) == 1 && !willHaveRegions,
		adaptiveManager: NewAdaptiveManager(dataStore),
		detailsManager:  NewDetailsPanelManager(),
	}

	m.dispatcher.AddHandler(m)
	m.dispatcher.AddHandler(m.adaptiveManager)

	m.startSSLCollection()
	return m
}

func (m *RefactoredManager) HandleUpdateEvent(event UpdateEvent) {
	switch event.Type {
	case TargetDataUpdateEvent:
		if !m.dataStore.ValidateDataConsistency(event.Key, event.Data) {
			return
		}
		m.dataStore.UpdateTargetData(event.Key, event.Data)
		m.dataStore.UpdatePlotData(event.Key, event.Data.Result, event.TermWidth)
		m.adaptiveManager.UpdateTargetStatus(event.Key, event.Data)

		currentTarget, currentRegion, currentKey := m.adaptiveManager.GetCurrentSelection()
		if currentKey == event.Key {
			m.updateCurrentDisplay(event.Data, currentTarget, currentRegion)
		} else {
			ui.Render(m.grid)
		}

	case SSLDataUpdateEvent:
		m.dataStore.UpdateSSLData(event.URL, event.SSLExpiry)
	}
}

func (m *RefactoredManager) startSSLCollection() {
	for _, target := range m.targets {
		go func(url string) {
			if strings.HasPrefix(url, "https://") {
				sslDaysRemaining := net.GetSSLCertExpiry(url)
				event := NewSSLUpdateEvent(url, sslDaysRemaining)
				m.dispatcher.DispatchEvent(event)
			}
		}(target.URL)
	}
}

func (m *RefactoredManager) getSSLExpiry(url string) int {
	if days, exists := m.dataStore.GetSSLData(url); exists {
		return days
	}
	return 0
}

func (m *RefactoredManager) InitializeLayout(width, height int) {
	m.termWidth = width
	m.termHeight = height

	if len(m.targets) > 0 {
		m.detailsManager.Initialize(m.targets[0].URL, m.targets[0].GetRefreshInterval())
	}

	m.adaptiveManager.Initialize(m.targets, m.hasRegions)

	m.setupGrid(width, height)

	if len(m.targets) > 0 {
		target, region, key := m.adaptiveManager.GetCurrentSelection()
		if key.TargetName != "" {
			m.detailsManager.UpdateTarget(target.URL, region, target.GetRefreshInterval())
		}
	}
}

func (m *RefactoredManager) setupGrid(width, height int) {
	m.grid = ui.NewGrid()
	m.grid.SetRect(0, 0, width, height)

	displayMode := m.adaptiveManager.GetDisplayMode()

	if displayMode == SinglePane {
		m.grid.Set(
			ui.NewRow(1.0, m.detailsManager.Grid),
		)
	} else {
		activeWidget := m.adaptiveManager.GetActiveWidget()
		if activeWidget != nil {
			m.grid.Set(
				ui.NewRow(1.0,
					ui.NewCol(0.25,
						ui.NewRow(1.0/7, m.adaptiveManager.GetSearchWidget()),
						ui.NewRow(6.0/7, activeWidget),
					),
					ui.NewCol(0.75, m.detailsManager.Grid),
				),
			)
		} else {
			m.grid.Set(
				ui.NewRow(1.0, m.detailsManager.Grid),
			)
		}
	}

	ui.Render(m.grid)
}

func (m *RefactoredManager) UpdateTarget(data TargetData) {
	var key TargetKey
	if data.Region != "" {
		key = NewRegionalTargetKey(data.Target, data.Region)
		m.hasRegions = true
		m.adaptiveManager.AddRegion(data.Target, data.Region)

		if m.isSingle {
			m.setupGrid(m.termWidth, m.termHeight)
		}
	} else {
		key = NewLocalTargetKey(data.Target)
	}

	event := NewTargetDataUpdateEvent(key, data, m.termWidth)
	m.dispatcher.DispatchEvent(event)
}

func (m *RefactoredManager) updateCurrentDisplay(data TargetData, target config.Target, region string) {
	m.detailsManager.UpdateTarget(target.URL, region, target.GetRefreshInterval())
	m.detailsManager.UpdateFromStats(data.Stats)

	sslExpiry := m.getSSLExpiry(data.Result.URL)
	m.detailsManager.UpdateFromResult(data.Result, sslExpiry)

	var plotKey TargetKey
	if data.Region != "" {
		plotKey = NewRegionalTargetKey(data.Target, data.Region)
	} else {
		plotKey = NewLocalTargetKey(data.Target)
	}

	m.restorePlotData(plotKey)

	ui.Render(m.grid)
}

func (m *RefactoredManager) SetActiveTarget(monitors map[string]*stats.Monitor) {
	target, region, key := m.adaptiveManager.GetCurrentSelection()
	if key.TargetName == "" {
		return
	}

	m.detailsManager.UpdateTarget(target.URL, region, target.GetRefreshInterval())

	if monitor, exists := monitors[key.String()]; exists {
		stats := monitor.GetStats()
		m.detailsManager.UpdateFromStats(stats)
	}

	if data, exists := m.dataStore.GetTargetData(key); exists {
		sslExpiry := m.getSSLExpiry(data.Result.URL)
		m.detailsManager.UpdateFromResult(data.Result, sslExpiry)
	}

	m.restorePlotData(key)
	ui.Render(m.grid)
}

func (m *RefactoredManager) RefreshStats(monitors map[string]*stats.Monitor) {
	_, _, key := m.adaptiveManager.GetCurrentSelection()
	if key.TargetName == "" {
		return
	}

	if monitor, exists := monitors[key.String()]; exists {
		stats := monitor.GetStats()
		m.detailsManager.UpdateFromStats(stats)
		ui.Render(m.grid)
	}
}

func (m *RefactoredManager) Navigate(direction int) {
	m.adaptiveManager.Navigate(direction)
	ui.Render(m.grid)
}

func (m *RefactoredManager) ToggleExpansion() bool {
	if m.adaptiveManager.ToggleExpansion() {
		ui.Render(m.grid)
		return true
	}
	return false
}

func (m *RefactoredManager) Resize(width, height int) {
	m.termWidth = width
	m.termHeight = height
	m.grid.SetRect(0, 0, width, height)
	m.detailsManager.Resize(width, height)
}

func (m *RefactoredManager) restorePlotData(targetKey TargetKey) {
	if history, exists := m.dataStore.GetPlotData(targetKey); exists {
		m.detailsManager.RestorePlots(history.UptimeData, history.ResponseTimeData)
	} else {
		m.detailsManager.ClearPlots()
	}
}

func (m *RefactoredManager) GetListWidget() any {
	return m.adaptiveManager.GetActiveWidget()
}

func (m *RefactoredManager) ToggleSearch() {
	m.adaptiveManager.ToggleSearch()
	m.setupGrid(m.termWidth, m.termHeight)
}

func (m *RefactoredManager) NavigateTargets(direction int, currentIndex *int, monitors map[string]*stats.Monitor) {
	m.Navigate(direction)
	m.SetActiveTarget(monitors)
}

func (m *RefactoredManager) GetGrid() *ui.Grid {
	return m.grid
}

func (m *RefactoredManager) GetListWidgetForSearch() *FilteredList {
	return m.adaptiveManager.GetListWidgetForSearch()
}
