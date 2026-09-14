package policy

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Policy defines sandbox rules for package installs.
type Policy struct {
	HidePaths          []string `yaml:"hide_paths"`
	HideEnvKeys        []string `yaml:"hide_env_keys"`
	AllowWritePaths    []string `yaml:"allow_write_paths"`
	AllowNetwork       bool     `yaml:"allow_network"`
	DryRun             bool     `yaml:"dry_run"`
	NpmWrapScripts     string   `yaml:"npm_wrap_scripts"`      // always|allowlisted|never; default always
	NpmScriptAllowlist []string `yaml:"npm_script_allowlist"` // reserved for future script-name filtering
}

// Default returns a sensible default policy hiding common secret locations.
func Default() Policy {
	home, _ := os.UserHomeDir()
	return Policy{
		HidePaths: []string{
			home + "/.ssh",
			home + "/.aws",
			home + "/.gnupg",
			".env",
		},
		HideEnvKeys: []string{
			"AWS_SECRET_ACCESS_KEY",
			"GITHUB_TOKEN",
			"NPM_TOKEN",
			"OPENAI_API_KEY",
		},
		AllowNetwork:   true,
		DryRun:         false,
		NpmWrapScripts: NpmWrapAlways,
	}
}

// Load reads a policy YAML file or returns defaults if path is empty.
func Load(path string) (Policy, error) {
	if path == "" {
		return Default(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("read policy: %w", err)
	}
	var p Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return Policy{}, fmt.Errorf("parse policy: %w", err)
	}
	// Merge with defaults for empty slices
	def := Default()
	if len(p.HidePaths) == 0 {
		p.HidePaths = def.HidePaths
	}
	if len(p.HideEnvKeys) == 0 {
		p.HideEnvKeys = def.HideEnvKeys
	}
	if p.NpmWrapScripts == "" {
		p.NpmWrapScripts = NpmWrapAlways
	}
	return p, nil
}
