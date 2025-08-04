package stats

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
			targetKey:   NewLocalTargetKey("api-server", -1),
			wantString:  "api-server",
			wantDisplay: "api-server",
		},
		{
			name:        "region target",
			targetKey:   NewRegionTargetKey("api-server", "us-east-1", -1),
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
			want:   NewLocalTargetKey("api-server", -1),
		},
		{
			name:   "name with region",
			keyStr: "api-server@us-west-2",
			want:   NewRegionTargetKey("api-server", "us-west-2", -1),
		},
		{
			name:   "name with multiple @ symbols",
			keyStr: "api@server@us-east-1",
			want:   NewRegionTargetKey("api", "server@us-east-1", -1),
		},
		{
			name:   "empty string",
			keyStr: "",
			want:   NewLocalTargetKey("", -1),
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
		index         int
		wantKeys      []TargetKey
	}{
		{
			name: "target with specific regions",
			target: config.Target{
				Name:    "api-server",
				Regions: []string{"us-east-1", "eu-west-1"},
			},
			globalRegions: []string{"us-west-2", "ap-south-1"},
			index:         0,
			wantKeys: []TargetKey{
				NewRegionTargetKey("api-server#0", "us-east-1", 0),
				NewRegionTargetKey("api-server#0", "eu-west-1", 0),
			},
		},
		{
			name: "target with no regions uses global",
			target: config.Target{
				Name:    "api-server",
				Regions: []string{},
			},
			globalRegions: []string{"us-east-1", "us-west-2"},
			index:         1,
			wantKeys: []TargetKey{
				NewRegionTargetKey("api-server#1", "us-east-1", 1),
				NewRegionTargetKey("api-server#1", "us-west-2", 1),
			},
		},
		{
			name: "target with no regions and no global regions",
			target: config.Target{
				Name:    "api-server",
				Regions: []string{},
			},
			globalRegions: []string{},
			index:         2,
			wantKeys: []TargetKey{
				NewLocalTargetKey("api-server#2", 2),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetAllKeysForTarget(tt.target, tt.globalRegions, tt.index)
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

func TestTargetKeyRegistryWithDuplicateNames(t *testing.T) {
	targets := []config.Target{
		{Name: "Test Service", URL: "https://httpbin.org/status/200", Regions: []string{}},
		{Name: "Test Service", URL: "https://httpbin.org/delay/2", Regions: []string{}},
		{Name: "Test Service", URL: "https://httpbin.org/status/404", Regions: []string{}},
		{Name: "Google", URL: "https://www.google.com", Regions: []string{}},
	}

	registry := NewTargetKeyRegistry(targets, []string{})

	t.Run("AllTargetsGetUniqueKeys", func(t *testing.T) {
		keys := registry.GetAllKeys()

		if len(keys) != 4 {
			t.Errorf("Expected 4 keys, got %d", len(keys))
		}

		expectedKeys := []string{
			"Test Service#0",
			"Test Service#1",
			"Test Service#2",
			"Google#3",
		}

		for i, key := range keys {
			if key.String() != expectedKeys[i] {
				t.Errorf("Key[%d] = %q, want %q", i, key.String(), expectedKeys[i])
			}
		}
	})

	t.Run("DuplicateNamesHaveSeparateKeys", func(t *testing.T) {
		allKeys := registry.GetAllKeys()
		keyStrings := make(map[string]int)

		for _, key := range allKeys {
			keyStrings[key.String()]++
		}

		for keyStr, count := range keyStrings {
			if count != 1 {
				t.Errorf("Key %q appears %d times, should be unique", keyStr, count)
			}
		}
	})
}

func TestTargetKeyWithSpecialCharacters(t *testing.T) {
	t.Run("TargetNamesWithAtSymbol", func(t *testing.T) {
		targets := []config.Target{
			{Name: "API@Service", URL: "https://api.example.com"},
			{Name: "DB@Server", URL: "https://db.example.com"},
		}

		registry := NewTargetKeyRegistry(targets, []string{"us-east-1"})
		keys := registry.GetAllKeys()

		if len(keys) != 2 {
			t.Errorf("Expected 2 keys, got %d", len(keys))
		}

		expectedKey1 := "API@Service#0@us-east-1"
		if keys[0].String() != expectedKey1 {
			t.Errorf("Key[0] = %q, want %q", keys[0].String(), expectedKey1)
		}

		expectedTargetName1 := "API@Service#0"
		if keys[0].TargetName != expectedTargetName1 {
			t.Errorf("TargetName[0] = %q, want %q", keys[0].TargetName, expectedTargetName1)
		}
	})

	t.Run("TargetNamesWithHashSymbol", func(t *testing.T) {
		targets := []config.Target{
			{Name: "Web#Service", URL: "https://web.example.com"},
			{Name: "Cache#Store#1", URL: "https://cache.example.com"},
		}

		registry := NewTargetKeyRegistry(targets, []string{"us-west-2"})
		keys := registry.GetAllKeys()

		if len(keys) != 2 {
			t.Errorf("Expected 2 keys, got %d", len(keys))
		}

		expectedKey1 := "Web#Service#0@us-west-2"
		if keys[0].String() != expectedKey1 {
			t.Errorf("Key[0] = %q, want %q", keys[0].String(), expectedKey1)
		}

		expectedKey2 := "Cache#Store#1#1@us-west-2"
		if keys[1].String() != expectedKey2 {
			t.Errorf("Key[1] = %q, want %q", keys[1].String(), expectedKey2)
		}
	})

	t.Run("ParseTargetKeyWithSpecialCharacters", func(t *testing.T) {
		tests := []struct {
			name           string
			input          string
			expectedTarget string
			expectedRegion string
		}{
			{
				name:           "@ symbol splits incorrectly",
				input:          "API@Service",
				expectedTarget: "API",
				expectedRegion: "Service",
			},
			{
				name:           "# symbol preserved in target name",
				input:          "Web#Service",
				expectedTarget: "Web#Service",
				expectedRegion: "local",
			},
			{
				name:           "multiple @ symbols - only first is delimiter",
				input:          "API@Service@Backend",
				expectedTarget: "API",
				expectedRegion: "Service@Backend",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				parsed := ParseTargetKey(tt.input)
				if parsed.TargetName != tt.expectedTarget {
					t.Errorf("TargetName = %q, want %q", parsed.TargetName, tt.expectedTarget)
				}
				if parsed.Region != tt.expectedRegion {
					t.Errorf("Region = %q, want %q", parsed.Region, tt.expectedRegion)
				}
			})
		}
	})
}
