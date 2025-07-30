package tui

import (
	"sync"

	"github.com/Owloops/updo/net"
)

type DataStore struct {
	targetDataMutex sync.RWMutex
	plotDataMutex   sync.RWMutex
	sslDataMutex    sync.RWMutex

	targetData map[TargetKey]TargetData
	plotData   map[TargetKey]PlotHistory
	sslData    map[string]int
}

func NewDataStore() *DataStore {
	return &DataStore{
		targetData: make(map[TargetKey]TargetData),
		plotData:   make(map[TargetKey]PlotHistory),
		sslData:    make(map[string]int),
	}
}

func (ds *DataStore) UpdateTargetData(key TargetKey, data TargetData) {
	if !ds.ValidateDataConsistency(key, data) {
		return
	}
	ds.targetDataMutex.Lock()
	defer ds.targetDataMutex.Unlock()
	ds.targetData[key] = data
}

func (ds *DataStore) GetTargetData(key TargetKey) (TargetData, bool) {
	ds.targetDataMutex.RLock()
	defer ds.targetDataMutex.RUnlock()
	data, exists := ds.targetData[key]
	return data, exists
}

func (ds *DataStore) UpdatePlotData(key TargetKey, result net.WebsiteCheckResult, termWidth int) {
	ds.plotDataMutex.Lock()
	defer ds.plotDataMutex.Unlock()

	history, exists := ds.plotData[key]
	if !exists {
		history = PlotHistory{
			UptimeData:       make([]float64, 0),
			ResponseTimeData: []float64{0.0, 0.0},
		}
	}

	if result.IsUp {
		history.UptimeData = append(history.UptimeData, 1.0)
	} else {
		history.UptimeData = append(history.UptimeData, 0.0)
	}
	history.ResponseTimeData = append(history.ResponseTimeData, result.ResponseTime.Seconds())

	maxLength := termWidth / 2
	if len(history.UptimeData) > maxLength {
		history.UptimeData = history.UptimeData[len(history.UptimeData)-maxLength:]
	}
	if len(history.ResponseTimeData) > maxLength {
		history.ResponseTimeData = history.ResponseTimeData[len(history.ResponseTimeData)-maxLength:]
	}

	ds.plotData[key] = history
}

func (ds *DataStore) GetPlotData(key TargetKey) (PlotHistory, bool) {
	ds.plotDataMutex.RLock()
	defer ds.plotDataMutex.RUnlock()
	history, exists := ds.plotData[key]
	return history, exists
}

func (ds *DataStore) UpdateSSLData(url string, daysRemaining int) {
	ds.sslDataMutex.Lock()
	defer ds.sslDataMutex.Unlock()
	ds.sslData[url] = daysRemaining
}

func (ds *DataStore) GetSSLData(url string) (int, bool) {
	ds.sslDataMutex.RLock()
	defer ds.sslDataMutex.RUnlock()
	days, exists := ds.sslData[url]
	return days, exists
}

func (ds *DataStore) GetAllTargetKeys() []TargetKey {
	ds.targetDataMutex.RLock()
	defer ds.targetDataMutex.RUnlock()

	keys := make([]TargetKey, 0, len(ds.targetData))
	for key := range ds.targetData {
		keys = append(keys, key)
	}
	return keys
}

func (ds *DataStore) ValidateDataConsistency(key TargetKey, data TargetData) bool {
	return key.TargetName == data.Target.Name && key.Region == data.Region
}
