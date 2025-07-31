package tui

import (
	"fmt"
	"strings"

	"github.com/Owloops/updo/config"
)

const localRegion = "local"

type TargetKey struct {
	TargetName string
	Region     string
	IsLocal    bool
}

func (tk TargetKey) String() string {
	if tk.IsLocal || tk.Region == "" || tk.Region == localRegion {
		return tk.TargetName
	}
	return fmt.Sprintf("%s@%s", tk.TargetName, tk.Region)
}

func (tk TargetKey) DisplayName() string {
	if tk.IsLocal || tk.Region == "" || tk.Region == localRegion {
		return tk.TargetName
	}
	return fmt.Sprintf("%s (%s)", tk.TargetName, tk.Region)
}

func NewTargetKey(targetName, region string) TargetKey {
	isLocal := region == "" || region == localRegion
	return TargetKey{
		TargetName: targetName,
		Region:     region,
		IsLocal:    isLocal,
	}
}

func NewLocalTargetKey(targetName string) TargetKey {
	return TargetKey{
		TargetName: targetName,
		Region:     localRegion,
		IsLocal:    true,
	}
}

func NewRegionTargetKey(targetName, region string) TargetKey {
	return TargetKey{
		TargetName: targetName,
		Region:     region,
		IsLocal:    false,
	}
}

func ParseTargetKey(keyStr string) TargetKey {
	if strings.Contains(keyStr, "@") {
		parts := strings.SplitN(keyStr, "@", 2)
		return NewRegionTargetKey(parts[0], parts[1])
	}
	return NewLocalTargetKey(keyStr)
}

func GetAllKeysForTarget(target config.Target, regions []string) []TargetKey {
	var keys []TargetKey

	targetRegions := target.Regions
	if len(targetRegions) == 0 {
		targetRegions = regions
	}

	if len(targetRegions) > 0 {
		for _, region := range targetRegions {
			keys = append(keys, NewRegionTargetKey(target.Name, region))
		}
	} else {
		keys = append(keys, NewLocalTargetKey(target.Name))
	}

	return keys
}

type TargetKeyRegistry struct {
	allKeys    []TargetKey
	keysByName map[string][]TargetKey
}

func NewTargetKeyRegistry(targets []config.Target, globalRegions []string) *TargetKeyRegistry {
	registry := &TargetKeyRegistry{
		allKeys:    make([]TargetKey, 0),
		keysByName: make(map[string][]TargetKey),
	}

	for _, target := range targets {
		targetKeys := GetAllKeysForTarget(target, globalRegions)
		registry.allKeys = append(registry.allKeys, targetKeys...)
		registry.keysByName[target.Name] = targetKeys
	}

	return registry
}

func (r *TargetKeyRegistry) GetAllKeys() []TargetKey {
	return r.allKeys
}

func (r *TargetKeyRegistry) GetKeysForTarget(targetName string) []TargetKey {
	if keys, exists := r.keysByName[targetName]; exists {
		return keys
	}
	return []TargetKey{}
}

func (r *TargetKeyRegistry) HasMultipleKeys() bool {
	for _, keys := range r.keysByName {
		if len(keys) > 1 {
			return true
		}
	}
	return false
}

func (r *TargetKeyRegistry) GetDisplayList() []string {
	displayList := make([]string, len(r.allKeys))
	for i, key := range r.allKeys {
		displayList[i] = key.DisplayName()
	}
	return displayList
}
