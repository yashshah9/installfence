package compat

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Entry is one package install to validate under installfence.
type Entry struct {
	ID         string   `yaml:"id"`
	Ecosystem  string   `yaml:"ecosystem"` // pip | npm | uv
	Package    string   `yaml:"package"`
	Version    string   `yaml:"version,omitempty"`
	Tags       []string `yaml:"tags,omitempty"`
	ExpectExit int      `yaml:"expect_exit"`
	PolicyFile string   `yaml:"policy,omitempty"`
}

// Matrix is the on-disk compatibility suite.
type Matrix struct {
	Version  int     `yaml:"version"`
	Packages []Entry `yaml:"packages"`
}

// Load reads a matrix YAML file.
func Load(path string) (Matrix, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Matrix{}, fmt.Errorf("read matrix: %w", err)
	}
	var m Matrix
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Matrix{}, fmt.Errorf("parse matrix: %w", err)
	}
	if err := m.Validate(); err != nil {
		return Matrix{}, err
	}
	return m, nil
}

// Validate checks required fields and ecosystems.
func (m Matrix) Validate() error {
	if len(m.Packages) == 0 {
		return fmt.Errorf("matrix has no packages")
	}
	seen := map[string]struct{}{}
	for i, e := range m.Packages {
		if e.ID == "" {
			return fmt.Errorf("packages[%d]: missing id", i)
		}
		if _, ok := seen[e.ID]; ok {
			return fmt.Errorf("duplicate package id %q", e.ID)
		}
		seen[e.ID] = struct{}{}
		if e.Package == "" {
			return fmt.Errorf("%s: missing package", e.ID)
		}
		switch strings.ToLower(e.Ecosystem) {
		case "pip", "npm", "uv":
		default:
			return fmt.Errorf("%s: unknown ecosystem %q (want pip|npm|uv)", e.ID, e.Ecosystem)
		}
	}
	return nil
}

// Filter returns entries matching all of ecosystems (if non-empty) and any of tags (if non-empty).
func (m Matrix) Filter(ecosystems, tags []string, limit int) []Entry {
	ecoSet := map[string]struct{}{}
	for _, e := range ecosystems {
		ecoSet[strings.ToLower(e)] = struct{}{}
	}
	tagSet := map[string]struct{}{}
	for _, t := range tags {
		tagSet[strings.ToLower(t)] = struct{}{}
	}

	var out []Entry
	for _, e := range m.Packages {
		if len(ecoSet) > 0 {
			if _, ok := ecoSet[strings.ToLower(e.Ecosystem)]; !ok {
				continue
			}
		}
		if len(tagSet) > 0 {
			match := false
			for _, t := range e.Tags {
				if _, ok := tagSet[strings.ToLower(t)]; ok {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		out = append(out, e)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

// InstallArgs builds the package-manager argv for an entry.
// workDir is a writable directory (typically under /tmp) used as --target/--prefix.
func InstallArgs(e Entry, workDir string) ([]string, error) {
	spec := e.Package
	if e.Version != "" {
		switch strings.ToLower(e.Ecosystem) {
		case "pip", "uv":
			spec = e.Package + "==" + e.Version
		case "npm":
			spec = e.Package + "@" + e.Version
		}
	}
	switch strings.ToLower(e.Ecosystem) {
	case "pip":
		return []string{
			"python3", "-m", "pip", "install",
			"--disable-pip-version-check", "--no-input",
			"--target", workDir,
			spec,
		}, nil
	case "uv":
		return []string{
			"uv", "pip", "install",
			"--target", workDir,
			spec,
		}, nil
	case "npm":
		cache := filepath.Join(workDir, ".npm-cache")
		return []string{
			"npm", "install",
			"--prefix", workDir,
			"--cache", cache,
			"--no-fund", "--no-audit", "--silent",
			spec,
		}, nil
	default:
		return nil, fmt.Errorf("unknown ecosystem %q", e.Ecosystem)
	}
}
