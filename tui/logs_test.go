package tui

import (
	"testing"

	"github.com/Owloops/updo/stats"
)

func TestLogBuffer(t *testing.T) {
	tests := []struct {
		name     string
		bufSize  int
		addCount int
		wantLen  int
	}{
		{
			name:     "buffer not full",
			bufSize:  10,
			addCount: 5,
			wantLen:  5,
		},
		{
			name:     "buffer exactly full",
			bufSize:  5,
			addCount: 5,
			wantLen:  5,
		},
		{
			name:     "buffer overflow",
			bufSize:  5,
			addCount: 10,
			wantLen:  5,
		},
		{
			name:     "single entry buffer",
			bufSize:  1,
			addCount: 5,
			wantLen:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := NewLogBuffer(tt.bufSize)

			for i := 0; i < tt.addCount; i++ {
				buffer.AddLogEntry(LogLevelInfo, "test", "message", stats.TargetKey{})
			}

			entries := buffer.GetEntries()
			if len(entries) != tt.wantLen {
				t.Errorf("GetEntries() returned %d entries, want %d", len(entries), tt.wantLen)
			}
		})
	}
}

func TestLogBuffer_CircularBehavior(t *testing.T) {
	buffer := NewLogBuffer(3)

	buffer.AddLogEntry(LogLevelInfo, "test", "msg1", stats.TargetKey{})
	buffer.AddLogEntry(LogLevelWarning, "test", "msg2", stats.TargetKey{})
	buffer.AddLogEntry(LogLevelError, "test", "msg3", stats.TargetKey{})
	buffer.AddLogEntry(LogLevelInfo, "test", "msg4", stats.TargetKey{})

	entries := buffer.GetEntries()
	if len(entries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(entries))
	}

	if entries[0].Details != "msg2" {
		t.Errorf("First entry should be msg2, got %s", entries[0].Details)
	}
	if entries[2].Details != "msg4" {
		t.Errorf("Last entry should be msg4, got %s", entries[2].Details)
	}
}

func TestLogBuffer_Clear(t *testing.T) {
	buffer := NewLogBuffer(5)

	buffer.AddLogEntry(LogLevelInfo, "test", "msg1", stats.TargetKey{})
	buffer.AddLogEntry(LogLevelInfo, "test", "msg2", stats.TargetKey{})

	if len(buffer.GetEntries()) != 2 {
		t.Error("Expected 2 entries before clear")
	}

	buffer.Clear()

	if len(buffer.GetEntries()) != 0 {
		t.Error("Expected 0 entries after clear")
	}
}

func TestLogBuffer_GetRecentEntries(t *testing.T) {
	buffer := NewLogBuffer(10)

	for i := 0; i < 10; i++ {
		buffer.AddLogEntry(LogLevelInfo, "test", "msg"+string(rune('0'+i)), stats.TargetKey{})
	}

	recent := buffer.GetRecentEntries(3)
	if len(recent) != 3 {
		t.Fatalf("GetRecentEntries(3) returned %d entries, want 3", len(recent))
	}

	if recent[0].Details != "msg9" {
		t.Errorf("Most recent entry should be msg9, got %s", recent[0].Details)
	}

	all := buffer.GetRecentEntries(20)
	if len(all) != 10 {
		t.Errorf("GetRecentEntries(20) should return all 10 entries, got %d", len(all))
	}

	zero := buffer.GetRecentEntries(0)
	if len(zero) != 10 {
		t.Errorf("GetRecentEntries(0) should return all entries, got %d", len(zero))
	}
}

func TestLogBuffer_GetEntriesForTarget(t *testing.T) {
	buffer := NewLogBuffer(10)

	target1 := stats.NewLocalTargetKey("api", 0)
	target2 := stats.NewRegionTargetKey("web", "us-east-1", 1)

	buffer.AddLogEntry(LogLevelInfo, "test", "msg1", target1)
	buffer.AddLogEntry(LogLevelInfo, "test", "msg2", target2)
	buffer.AddLogEntry(LogLevelError, "test", "msg3", target1)
	buffer.AddLogEntry(LogLevelWarning, "test", "msg4", target2)

	target1Entries := buffer.GetEntriesForTarget(target1)
	if len(target1Entries) != 2 {
		t.Errorf("Expected 2 entries for target1, got %d", len(target1Entries))
	}

	target2Entries := buffer.GetEntriesForTarget(target2)
	if len(target2Entries) != 2 {
		t.Errorf("Expected 2 entries for target2, got %d", len(target2Entries))
	}

	emptyTarget := stats.NewLocalTargetKey("empty", 2)
	emptyEntries := buffer.GetEntriesForTarget(emptyTarget)
	if len(emptyEntries) != 0 {
		t.Errorf("Expected 0 entries for empty target, got %d", len(emptyEntries))
	}
}

func TestLogBuffer_EdgeCases(t *testing.T) {
	t.Run("zero size buffer", func(t *testing.T) {
		buffer := NewLogBuffer(0)
		if buffer.MaxSize() != 100 {
			t.Errorf("Zero size should default to 100, got %d", buffer.MaxSize())
		}
	})

	t.Run("negative size buffer", func(t *testing.T) {
		buffer := NewLogBuffer(-10)
		if buffer.MaxSize() != 100 {
			t.Errorf("Negative size should default to 100, got %d", buffer.MaxSize())
		}
	})

	t.Run("concurrent access", func(t *testing.T) {
		buffer := NewLogBuffer(100)
		done := make(chan bool)

		go func() {
			for i := 0; i < 50; i++ {
				buffer.AddLogEntry(LogLevelInfo, "writer1", "msg", stats.TargetKey{})
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 50; i++ {
				buffer.AddLogEntry(LogLevelInfo, "writer2", "msg", stats.TargetKey{})
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 100; i++ {
				_ = buffer.GetEntries()
				_ = buffer.Size()
			}
			done <- true
		}()

		for i := 0; i < 3; i++ {
			<-done
		}

		if buffer.Size() != 100 {
			t.Errorf("Expected 100 entries after concurrent writes, got %d", buffer.Size())
		}
	})
}
