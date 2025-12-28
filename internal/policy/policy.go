package policy

import (
	"embed"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"
)

//go:embed default_policy.yaml
var defaultPolicyData []byte

// Thresholds controls preview and risk limits.
type Thresholds struct {
	DeleteCount  int `yaml:"delete_count"`
	ModifyCount  int `yaml:"modify_count"`
	PreviewSample int `yaml:"preview_sample"`
}

// ArbiterConfig describes optional LLM arbiter settings.
type ArbiterConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Provider   string `yaml:"provider"`
	Model      string `yaml:"model"`
	APIKeyEnv  string `yaml:"api_key_env"`
}

// Options holds miscellaneous toggles.
type Options struct {
	AllowOutsideRepo              bool `yaml:"allow_outside_repo"`
	RequireCleanTreeForBreakGlass bool `yaml:"require_clean_tree_for_break_glass"`
}

// Policy represents the effective ruleset.
type Policy struct {
	Thresholds      Thresholds   `yaml:"thresholds"`
	ProtectedPaths  []string     `yaml:"protected_paths"`
	AllowCommands   []string     `yaml:"allow_commands"`
	BlockCommands   []string     `yaml:"block_commands"`
	ConfirmCommands []string     `yaml:"confirm_commands"`
	NetworkEgress   []string     `yaml:"network_egress"`
	PackageManagers []string     `yaml:"package_managers"`
	Arbiter         ArbiterConfig `yaml:"arbiter"`
	Options         Options      `yaml:"options"`
}

// Load returns the effective policy, merging defaults with a repo-local file if present.
func Load(path string) (Policy, error) {
	base, err := parse(defaultPolicyData)
	if err != nil {
		return Policy{}, fmt.Errorf("parse default policy: %w", err)
	}

	if path == "" {
		return base, nil
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return base, nil
		}
		return base, fmt.Errorf("stat policy: %w", err)
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return base, fmt.Errorf("read policy: %w", err)
	}
	user, err := parse(data)
	if err != nil {
		return base, fmt.Errorf("parse policy: %w", err)
	}

	merge(&base, user)
	return base, nil
}

func parse(data []byte) (Policy, error) {
	var p Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return Policy{}, err
	}
	return p, nil
}

func merge(base *Policy, override Policy) {
	if override.Thresholds.DeleteCount != 0 {
		base.Thresholds.DeleteCount = override.Thresholds.DeleteCount
	}
	if override.Thresholds.ModifyCount != 0 {
		base.Thresholds.ModifyCount = override.Thresholds.ModifyCount
	}
	if override.Thresholds.PreviewSample != 0 {
		base.Thresholds.PreviewSample = override.Thresholds.PreviewSample
	}

	if len(override.ProtectedPaths) > 0 {
		base.ProtectedPaths = override.ProtectedPaths
	}
	if len(override.AllowCommands) > 0 {
		base.AllowCommands = override.AllowCommands
	}
	if len(override.BlockCommands) > 0 {
		base.BlockCommands = override.BlockCommands
	}
	if len(override.ConfirmCommands) > 0 {
		base.ConfirmCommands = override.ConfirmCommands
	}
	if len(override.NetworkEgress) > 0 {
		base.NetworkEgress = override.NetworkEgress
	}
	if len(override.PackageManagers) > 0 {
		base.PackageManagers = override.PackageManagers
	}

	if override.Arbiter.Enabled {
		base.Arbiter = override.Arbiter
	} else {
		if override.Arbiter.Provider != "" || override.Arbiter.Model != "" || override.Arbiter.APIKeyEnv != "" {
			base.Arbiter = override.Arbiter
		}
	}

	base.Options.AllowOutsideRepo = base.Options.AllowOutsideRepo || override.Options.AllowOutsideRepo
	if override.Options.RequireCleanTreeForBreakGlass {
		base.Options.RequireCleanTreeForBreakGlass = true
	}
}

// ToYAML renders the policy to YAML.
func (p Policy) ToYAML() (string, error) {
	out, err := yaml.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// DefaultYAML returns the embedded default policy YAML.
func DefaultYAML() string {
	return string(defaultPolicyData)
}
