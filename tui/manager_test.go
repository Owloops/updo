package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/Owloops/updo/config"
	"github.com/Owloops/updo/net"
	"github.com/Owloops/updo/stats"
)

func TestNewManager(t *testing.T) {
	targets := []config.Target{
		{Name: "api", URL: "https://api.example.com"},
		{Name: "web", URL: "https://web.example.com"},
	}

	manager := NewManager(targets, Options{Regions: []string{}})

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.keyRegistry == nil {
		t.Error("keyRegistry is nil")
	}

	if manager.detailsManager == nil {
		t.Error("detailsManager is nil")
	}

	if manager.logBuffer == nil {
		t.Error("logBuffer is nil")
	}

	if manager.currentKeyIndex != 0 {
		t.Errorf("currentKeyIndex = %d, want 0", manager.currentKeyIndex)
	}
}

func TestManager_UpdateActiveTarget(t *testing.T) {
	targets := []config.Target{
		{Name: "test", URL: "https://example.com"},
	}

	manager := NewManager(targets, Options{Regions: []string{}})

	if manager.currentKeyIndex < 0 {
		t.Error("currentKeyIndex should not be negative")
	}

	allKeys := manager.keyRegistry.GetAllKeys()
	if manager.currentKeyIndex >= len(allKeys) {
		t.Error("currentKeyIndex should be within bounds")
	}

	currentKey := allKeys[manager.currentKeyIndex]
	if currentKey.TargetName == "" {
		t.Error("Current key should have a target name")
	}
}

func TestManager_RefreshStats(t *testing.T) {
	targets := []config.Target{
		{Name: "test", URL: "https://example.com"},
	}

	manager := NewManager(targets, Options{Regions: []string{}})

	monitor := &stats.Monitor{}
	monitor.AddResult(net.WebsiteCheckResult{
		IsUp:         true,
		ResponseTime: 100 * time.Millisecond,
		StatusCode:   200,
	})

	monitors := map[string]*stats.Monitor{
		"test#0": monitor,
	}

	allKeys := manager.keyRegistry.GetAllKeys()
	if len(allKeys) == 0 {
		t.Fatal("Should have at least one key")
	}

	if manager.currentKeyIndex < 0 || manager.currentKeyIndex >= len(allKeys) {
		t.Error("currentKeyIndex out of bounds")
	}

	currentKey := allKeys[manager.currentKeyIndex]
	if _, exists := monitors[currentKey.String()]; !exists {
		t.Error("Monitor should exist for current key")
	}
}

func TestManager_SingleTarget(t *testing.T) {
	targets := []config.Target{
		{Name: "single", URL: "https://single.example.com"},
	}

	manager := NewManager(targets, Options{Regions: []string{}})

	if !manager.isSingle {
		t.Error("Manager should be in single target mode")
	}
}

func TestManager_MultipleTargets(t *testing.T) {
	targets := []config.Target{
		{Name: "api", URL: "https://api.example.com"},
		{Name: "web", URL: "https://web.example.com"},
		{Name: "db", URL: "https://db.example.com"},
	}

	manager := NewManager(targets, Options{Regions: []string{"us-east-1", "us-west-2"}})

	if manager.isSingle {
		t.Error("Manager should not be in single target mode")
	}

	keys := manager.keyRegistry.GetAllKeys()
	if len(keys) == 0 {
		t.Error("Should have target keys")
	}

	expectedMinKeys := len(targets) * len([]string{"us-east-1", "us-west-2"})
	if len(keys) < expectedMinKeys {
		t.Errorf("Expected at least %d keys, got %d", expectedMinKeys, len(keys))
	}

	allKeys := manager.keyRegistry.GetAllKeys()
	if manager.currentKeyIndex >= len(allKeys) {
		t.Error("currentKeyIndex should be within bounds")
	}
}

func TestManager_LogBuffer(t *testing.T) {
	targets := []config.Target{
		{Name: "test", URL: "https://example.com"},
	}

	manager := NewManager(targets, Options{Regions: []string{}})

	if manager.logBuffer.Size() != 0 {
		t.Error("Log buffer should be empty initially")
	}

	manager.logBuffer.AddLogEntry(LogLevelInfo, "test", "Test message", stats.NewLocalTargetKey("test", 0))

	if manager.logBuffer.Size() != 1 {
		t.Errorf("Log buffer size = %d, want 1", manager.logBuffer.Size())
	}

	entries := manager.logBuffer.GetEntries()
	if len(entries) != 1 {
		t.Errorf("GetEntries returned %d entries, want 1", len(entries))
	}

	if entries[0].Message != "test" {
		t.Errorf("Log entry message = %q, want 'test'", entries[0].Message)
	}
}

func TestManager_KeyNavigation(t *testing.T) {
	targets := []config.Target{
		{Name: "target1", URL: "https://target1.com"},
		{Name: "target2", URL: "https://target2.com"},
		{Name: "target3", URL: "https://target3.com"},
	}

	manager := NewManager(targets, Options{Regions: []string{}})
	allKeys := manager.keyRegistry.GetAllKeys()

	if len(allKeys) != 3 {
		t.Fatalf("Expected 3 keys, got %d", len(allKeys))
	}

	if manager.currentKeyIndex < 0 || manager.currentKeyIndex >= len(allKeys) {
		t.Errorf("currentKeyIndex %d is out of bounds [0, %d)", manager.currentKeyIndex, len(allKeys))
	}

	currentKey := allKeys[manager.currentKeyIndex]
	if currentKey.TargetName == "" {
		t.Error("Current key has empty target name")
	}
}

func TestManager_ToggleLogsVisibility(t *testing.T) {
	targets := []config.Target{
		{Name: "test", URL: "https://example.com"},
	}

	manager := NewManager(targets, Options{Regions: []string{}})

	if manager.showLogs {
		t.Error("Logs should not be visible initially")
	}

	if manager.focusOnLogs {
		t.Error("Focus should not be on logs initially")
	}

	manager.showLogs = true
	manager.focusOnLogs = true

	if !manager.showLogs {
		t.Error("Logs should be visible after manual toggle")
	}

	if !manager.focusOnLogs {
		t.Error("Focus should be on logs after manual toggle")
	}
}

func TestManager_HeaderDetection(t *testing.T) {
	testRows := []string{
		"▼ api",
		"  ◉ us-east-1",
		"▶ web",
		"  ◉ us-east-1",
	}

	for i, row := range testRows {
		isHeader := strings.HasPrefix(row, "▼") || strings.HasPrefix(row, "▶")
		expectedHeader := i == 0 || i == 2

		if isHeader != expectedHeader {
			t.Errorf("Row %d (%q): header detection got %v, expected %v", i, row, isHeader, expectedHeader)
		}
	}
}
