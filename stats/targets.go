package stats

import (
	"fmt"
	"strings"

	"github.com/Owloops/updo/config"
)

const (
	_localRegion = "local"
)

type TargetKey struct {
	TargetName  string
	Region      string
	IsLocal     bool
	TargetIndex int
}

func (tk TargetKey) String() string {
	if tk.IsLocal || tk.Region == "" || tk.Region == _localRegion {
		return tk.TargetName
	}
	return fmt.Sprintf("%s@%s", tk.TargetName, tk.Region)
}

func (tk TargetKey) DisplayName() string {
	cleanName := tk.GetCleanName()
	if tk.IsLocal || tk.Region == "" || tk.Region == _localRegion {
		return cleanName
	}
	return fmt.Sprintf("%s (%s)", cleanName, tk.Region)
}

func (tk TargetKey) GetCleanName() string {
	if idx := strings.LastIndex(tk.TargetName, "#"); idx != -1 {
		return tk.TargetName[:idx]
	}
	return tk.TargetName
}

func NewTargetKey(targetName, region string) TargetKey {
	isLocal := region == "" || region == _localRegion
	return TargetKey{
		TargetName: targetName,
		Region:     region,
		IsLocal:    isLocal,
	}
}

func NewLocalTargetKey(targetName string, targetIndex int) TargetKey {
	return TargetKey{
		TargetName:  targetName,
		Region:      _localRegion,
		IsLocal:     true,
		TargetIndex: targetIndex,
	}
}

func NewRegionTargetKey(targetName, region string, targetIndex int) TargetKey {
	return TargetKey{
		TargetName:  targetName,
		Region:      region,
		IsLocal:     false,
		TargetIndex: targetIndex,
	}
}

func ParseTargetKey(keyStr string) TargetKey {
	if strings.Contains(keyStr, "@") {
		parts := strings.SplitN(keyStr, "@", 2)
		return NewRegionTargetKey(parts[0], parts[1], -1)
	}
	return NewLocalTargetKey(keyStr, -1)
}

func GetAllKeysForTarget(target config.Target, regions []string, index int) []TargetKey {
	var keys []TargetKey

	uniqueName := fmt.Sprintf("%s#%d", target.Name, index)

	targetRegions := target.Regions
	if len(targetRegions) == 0 {
		targetRegions = regions
	}

	if len(targetRegions) > 0 {
		for _, region := range targetRegions {
			keys = append(keys, NewRegionTargetKey(uniqueName, region, index))
		}
	} else {
		keys = append(keys, NewLocalTargetKey(uniqueName, index))
	}

	return keys
}

type TargetKeyRegistry struct {
	allKeys    []TargetKey
	keysByName map[string][]TargetKey
}

func NewTargetKeyRegistry(targets []config.Target, globalRegions []string) *TargetKeyRegistry {
	registry := &TargetKeyRegistry{
		keysByName: make(map[string][]TargetKey, len(targets)),
	}

	for i, target := range targets {
		targetKeys := GetAllKeysForTarget(target, globalRegions, i)
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
	return nil
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
