package shim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallWritesWrappers(t *testing.T) {
	dir := t.TempDir()
	if err := Install(dir, "/usr/local/bin/installfence", "/tmp/policy.yaml"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "pip"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "/usr/local/bin/installfence") {
		t.Fatalf("missing binary path: %s", text)
	}
	if !strings.Contains(text, " run -- pip ") {
		t.Fatalf("shim should wrap pip: %s", text)
	}
	if !strings.Contains(text, "--policy /tmp/policy.yaml") {
		t.Fatalf("missing policy flag: %s", text)
	}
}

func TestUninstallRemovesWrappers(t *testing.T) {
	dir := t.TempDir()
	if err := Install(dir, "/bin/installfence", ""); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "npm")); !os.IsNotExist(err) {
		t.Fatal("expected npm shim to be removed")
	}
}

func TestStatusDetectsPATH(t *testing.T) {
	dir := t.TempDir()
	infos := Status(dir, dir+string(os.PathListSeparator)+"/usr/bin")
	if len(infos) != len(Tools) {
		t.Fatalf("expected %d entries", len(Tools))
	}
	if !infos[0].OnPATH || !infos[0].BeforeOS {
		t.Fatalf("shim dir should be first on PATH: %+v", infos[0])
	}
}

func TestEnvSnippet(t *testing.T) {
	zsh := EnvSnippet("/tmp/shims", "zsh")
	if !strings.Contains(zsh, `export PATH="/tmp/shims":$PATH`) {
		t.Fatalf("unexpected zsh snippet: %s", zsh)
	}
	fish := EnvSnippet("/tmp/shims", "fish")
	if !strings.Contains(fish, "set -gx PATH /tmp/shims $PATH") {
		t.Fatalf("unexpected fish snippet: %s", fish)
	}
}
