package policy_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yashshah9/installfence/internal/policy"
)

func TestDefaultPolicyHasHidePaths(t *testing.T) {
	p := policy.Default()
	if len(p.HidePaths) == 0 {
		t.Fatal("expected default hide paths")
	}
}

func TestLoadPolicyFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	content := "allow_network: false\nhide_paths:\n  - /tmp/secret\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := policy.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.AllowNetwork {
		t.Fatal("expected allow_network false")
	}
}

func TestLoadPolicyAllowWritePaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	content := "allow_write_paths:\n  - " + dir + "\n  - /nonexistent/path\nallow_network: true\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := policy.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.AllowWritePaths) != 2 {
		t.Fatalf("expected 2 allow_write_paths, got %v", p.AllowWritePaths)
	}
	if p.AllowWritePaths[0] != dir {
		t.Fatalf("unexpected first path: %q", p.AllowWritePaths[0])
	}
}

func TestLoadPolicyEmptyPathUsesDefaults(t *testing.T) {
	p, err := policy.Load("")
	if err != nil {
		t.Fatal(err)
	}
	def := policy.Default()
	if len(p.HidePaths) != len(def.HidePaths) {
		t.Fatalf("expected default hide paths, got %v", p.HidePaths)
	}
}

func TestLoadPolicyMergesEmptyHidePathsWithDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	content := "allow_network: false\nallow_write_paths:\n  - /tmp/writable\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := policy.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.HidePaths) == 0 {
		t.Fatal("expected hide_paths merged from defaults")
	}
	if len(p.AllowWritePaths) != 1 || p.AllowWritePaths[0] != "/tmp/writable" {
		t.Fatalf("unexpected allow_write_paths: %v", p.AllowWritePaths)
	}
}
