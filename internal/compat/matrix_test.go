package compat

import (
	"path/filepath"
	"runtime"
	"testing"
)

func matrixPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "config", "compat-matrix.yaml")
}

func TestLoadMatrix(t *testing.T) {
	m, err := Load(matrixPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Packages) < 100 {
		t.Fatalf("want >=100 packages, got %d", len(m.Packages))
	}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestFilterSmoke(t *testing.T) {
	m, err := Load(matrixPath(t))
	if err != nil {
		t.Fatal(err)
	}
	smoke := m.Filter(nil, []string{"smoke"}, 0)
	if len(smoke) < 10 {
		t.Fatalf("smoke tags too few: %d", len(smoke))
	}
	pipOnly := m.Filter([]string{"pip"}, []string{"smoke"}, 0)
	for _, e := range pipOnly {
		if e.Ecosystem != "pip" {
			t.Fatalf("expected pip, got %s", e.Ecosystem)
		}
	}
}

func TestInstallArgs(t *testing.T) {
	args, err := InstallArgs(Entry{Ecosystem: "pip", Package: "requests", Version: "2.32.3"}, "/tmp/w")
	if err != nil {
		t.Fatal(err)
	}
	joined := joinArgs(args)
	if joined == "" || args[0] != "python3" {
		t.Fatalf("unexpected args: %v", args)
	}
	if joinArgs(args) != "python3 -m pip install --disable-pip-version-check --no-input --target /tmp/w requests==2.32.3" {
		t.Fatalf("pip args: %s", joined)
	}

	npm, err := InstallArgs(Entry{Ecosystem: "npm", Package: "lodash"}, "/tmp/n")
	if err != nil {
		t.Fatal(err)
	}
	if npm[0] != "npm" || npm[len(npm)-1] != "lodash" {
		t.Fatalf("npm args: %v", npm)
	}
}

func TestRunDryRunSmoke(t *testing.T) {
	m, err := Load(matrixPath(t))
	if err != nil {
		t.Fatal(err)
	}
	entries := m.Filter(nil, []string{"smoke"}, 5)
	rep, err := Run(entries, RunOptions{Mode: ModeDryRun})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Failed != 0 || rep.Passed != len(entries) {
		t.Fatalf("dry-run report: %+v", rep)
	}
}
