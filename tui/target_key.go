package tui

import (
	"fmt"
	"strings"

	"github.com/Owloops/updo/config"
)

type TargetKey struct {
	TargetName string
	Region     string
}

func (tk TargetKey) String() string {
	if tk.Region == "" {
		return tk.TargetName
	}
	return fmt.Sprintf("%s@%s", tk.TargetName, tk.Region)
}

func (tk TargetKey) IsLocal() bool {
	return tk.Region == ""
}

func (tk TargetKey) IsRegional() bool {
	return tk.Region != ""
}

func (tk TargetKey) Validate() error {
	if tk.TargetName == "" {
		return fmt.Errorf("target name cannot be empty")
	}
	return nil
}

func NewTargetKey(target config.Target, region string) TargetKey {
	return TargetKey{
		TargetName: target.Name,
		Region:     region,
	}
}

func NewLocalTargetKey(target config.Target) TargetKey {
	return TargetKey{
		TargetName: target.Name,
		Region:     "",
	}
}

func NewRegionalTargetKey(target config.Target, region string) TargetKey {
	return TargetKey{
		TargetName: target.Name,
		Region:     region,
	}
}

func ParseTargetKey(key string) (TargetKey, error) {
	if key == "" {
		return TargetKey{}, fmt.Errorf("key cannot be empty")
	}

	atIndex := strings.LastIndex(key, "@")
	if atIndex == -1 {
		return TargetKey{
			TargetName: key,
			Region:     "",
		}, nil
	}

	targetName := key[:atIndex]
	region := key[atIndex+1:]

	if targetName == "" {
		return TargetKey{}, fmt.Errorf("target name cannot be empty in key %q", key)
	}
	if region == "" {
		return TargetKey{}, fmt.Errorf("region cannot be empty in key %q", key)
	}

	return TargetKey{
		TargetName: targetName,
		Region:     region,
	}, nil
}
