package policy_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/installfence/installfence/internal/policy"
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
