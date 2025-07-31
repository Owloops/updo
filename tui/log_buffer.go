package tui

import (
	"sync"
	"time"
)

type LogLevel string

const (
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
)

type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
	Details   string
	TargetKey TargetKey
}

type LogBuffer struct {
	entries []LogEntry
	head    int
	size    int
	maxSize int
	full    bool
	mu      sync.RWMutex
}

func NewLogBuffer(maxSize int) *LogBuffer {
	if maxSize <= 0 {
		maxSize = 100
	}
	return &LogBuffer{
		entries: make([]LogEntry, maxSize),
		maxSize: maxSize,
	}
}

func (lb *LogBuffer) Add(entry LogEntry) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.entries[lb.head] = entry
	lb.head = (lb.head + 1) % lb.maxSize

	if !lb.full {
		lb.size++
		if lb.size == lb.maxSize {
			lb.full = true
		}
	}
}

func (lb *LogBuffer) GetEntries() []LogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if lb.size == 0 {
		return []LogEntry{}
	}

	result := make([]LogEntry, lb.size)

	if lb.full {
		copy(result, lb.entries[lb.head:])
		copy(result[lb.maxSize-lb.head:], lb.entries[:lb.head])
	} else {
		copy(result, lb.entries[:lb.size])
	}

	return result
}

func (lb *LogBuffer) GetRecentEntries(n int) []LogEntry {
	entries := lb.GetEntries()

	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	if n > 0 && n < len(entries) {
		return entries[:n]
	}
	return entries
}

func (lb *LogBuffer) Size() int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.size
}

func (lb *LogBuffer) Clear() {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.head = 0
	lb.size = 0
	lb.full = false
}

func (lb *LogBuffer) IsEmpty() bool {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.size == 0
}

func (lb *LogBuffer) MaxSize() int {
	return lb.maxSize
}

func (lb *LogBuffer) GetEntriesForTarget(targetKey TargetKey) []LogEntry {
	entries := lb.GetEntries()
	var filtered []LogEntry

	for _, entry := range entries {
		if entry.TargetKey.String() == targetKey.String() {
			filtered = append(filtered, entry)
		}
	}

	return filtered
}

func (lb *LogBuffer) AddLogEntry(level LogLevel, message, details string, targetKey TargetKey) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Details:   details,
		TargetKey: targetKey,
	}
	lb.Add(entry)
}
