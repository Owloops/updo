package tui

import (
	"reflect"
	"testing"

	"github.com/Owloops/updo/config"
)

func TestTargetKey(t *testing.T) {
	tests := []struct {
		name        string
		targetKey   TargetKey
		wantString  string
		wantDisplay string
	}{
		{
			name:        "local target",
			targetKey:   NewLocalTargetKey("api-server"),
			wantString:  "api-server",
			wantDisplay: "api-server",
		},
		{
			name:        "region target",
			targetKey:   NewRegionTargetKey("api-server", "us-east-1"),
			wantString:  "api-server@us-east-1",
			wantDisplay: "api-server (us-east-1)",
		},
		{
			name:        "empty region treated as local",
			targetKey:   NewTargetKey("api-server", ""),
			wantString:  "api-server",
			wantDisplay: "api-server",
		},
		{
			name:        "local region treated as local",
			targetKey:   NewTargetKey("api-server", "local"),
			wantString:  "api-server",
			wantDisplay: "api-server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.targetKey.String(); got != tt.wantString {
				t.Errorf("String() = %v, want %v", got, tt.wantString)
			}
			if got := tt.targetKey.DisplayName(); got != tt.wantDisplay {
				t.Errorf("DisplayName() = %v, want %v", got, tt.wantDisplay)
			}
		})
	}
}

func TestParseTargetKey(t *testing.T) {
	tests := []struct {
		name   string
		keyStr string
		want   TargetKey
	}{
		{
			name:   "simple name",
			keyStr: "api-server",
			want:   NewLocalTargetKey("api-server"),
		},
		{
			name:   "name with region",
			keyStr: "api-server@us-west-2",
			want:   NewRegionTargetKey("api-server", "us-west-2"),
		},
		{
			name:   "name with multiple @ symbols",
			keyStr: "api@server@us-east-1",
			want:   NewRegionTargetKey("api", "server@us-east-1"),
		},
		{
			name:   "empty string",
			keyStr: "",
			want:   NewLocalTargetKey(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTargetKey(tt.keyStr)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseTargetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAllKeysForTarget(t *testing.T) {
	tests := []struct {
		name          string
		target        config.Target
		globalRegions []string
		wantKeys      []TargetKey
	}{
		{
			name: "target with specific regions",
			target: config.Target{
				Name:    "api-server",
				Regions: []string{"us-east-1", "eu-west-1"},
			},
			globalRegions: []string{"us-west-2", "ap-south-1"},
			wantKeys: []TargetKey{
				NewRegionTargetKey("api-server", "us-east-1"),
				NewRegionTargetKey("api-server", "eu-west-1"),
			},
		},
		{
			name: "target with no regions uses global",
			target: config.Target{
				Name:    "api-server",
				Regions: []string{},
			},
			globalRegions: []string{"us-east-1", "us-west-2"},
			wantKeys: []TargetKey{
				NewRegionTargetKey("api-server", "us-east-1"),
				NewRegionTargetKey("api-server", "us-west-2"),
			},
		},
		{
			name: "target with no regions and no global regions",
			target: config.Target{
				Name:    "api-server",
				Regions: []string{},
			},
			globalRegions: []string{},
			wantKeys: []TargetKey{
				NewLocalTargetKey("api-server"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetAllKeysForTarget(tt.target, tt.globalRegions)
			if !reflect.DeepEqual(got, tt.wantKeys) {
				t.Errorf("GetAllKeysForTarget() = %v, want %v", got, tt.wantKeys)
			}
		})
	}
}

func TestTargetKeyRegistry(t *testing.T) {
	targets := []config.Target{
		{Name: "api", Regions: []string{"us-east-1", "us-west-2"}},
		{Name: "web", Regions: []string{"eu-west-1"}},
		{Name: "local-only", Regions: []string{}},
	}
	globalRegions := []string{"ap-south-1"}

	registry := NewTargetKeyRegistry(targets, globalRegions)

	t.Run("GetAllKeys", func(t *testing.T) {
		keys := registry.GetAllKeys()
		expectedCount := 4
		if len(keys) != expectedCount {
			t.Errorf("GetAllKeys() returned %d keys, want %d", len(keys), expectedCount)
		}
	})

	t.Run("GetKeysForTarget", func(t *testing.T) {
		tests := []struct {
			targetName string
			wantCount  int
		}{
			{"api", 2},
			{"web", 1},
			{"local-only", 1},
			{"non-existent", 0},
		}

		for _, tt := range tests {
			keys := registry.GetKeysForTarget(tt.targetName)
			if len(keys) != tt.wantCount {
				t.Errorf("GetKeysForTarget(%q) returned %d keys, want %d",
					tt.targetName, len(keys), tt.wantCount)
			}
		}
	})

	t.Run("HasMultipleKeys", func(t *testing.T) {
		if !registry.HasMultipleKeys() {
			t.Error("HasMultipleKeys() = false, want true")
		}

		singleKeyRegistry := NewTargetKeyRegistry(
			[]config.Target{{Name: "single", Regions: []string{}}},
			[]string{},
		)
		if singleKeyRegistry.HasMultipleKeys() {
			t.Error("HasMultipleKeys() = true for single key registry, want false")
		}
	})

	t.Run("GetDisplayList", func(t *testing.T) {
		displayList := registry.GetDisplayList()
		if len(displayList) != len(registry.GetAllKeys()) {
			t.Errorf("GetDisplayList() length = %d, want %d",
				len(displayList), len(registry.GetAllKeys()))
		}

		for i, display := range displayList {
			if display == "" {
				t.Errorf("GetDisplayList()[%d] is empty", i)
			}
		}
	})
}
