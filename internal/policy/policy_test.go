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

func TestLoadPolicyNpmWrapScripts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	content := "npm_wrap_scripts: never\nnpm_script_allowlist:\n  - esbuild\n  - sharp\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := policy.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.NpmWrapScripts != policy.NpmWrapNever {
		t.Fatalf("expected never, got %q", p.NpmWrapScripts)
	}
	if len(p.NpmScriptAllowlist) != 2 || p.NpmScriptAllowlist[0] != "esbuild" {
		t.Fatalf("unexpected allowlist: %v", p.NpmScriptAllowlist)
	}
}

func TestLoadPolicyNpmWrapScriptsDefaultsToAlways(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	content := "allow_network: true\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := policy.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.NpmWrapScripts != policy.NpmWrapAlways {
		t.Fatalf("expected always default, got %q", p.NpmWrapScripts)
	}
}

func TestApplyNpmWrapScripts(t *testing.T) {
	tests := []struct {
		name string
		p    policy.Policy
		args []string
		want []string
	}{
		{
			name: "always leaves install alone",
			p:    policy.Policy{NpmWrapScripts: policy.NpmWrapAlways},
			args: []string{"npm", "install"},
			want: []string{"npm", "install"},
		},
		{
			name: "never appends ignore-scripts",
			p:    policy.Policy{NpmWrapScripts: policy.NpmWrapNever},
			args: []string{"npm", "install", "lodash"},
			want: []string{"npm", "install", "lodash", "--ignore-scripts"},
		},
		{
			name: "never does not duplicate ignore-scripts",
			p:    policy.Policy{NpmWrapScripts: policy.NpmWrapNever},
			args: []string{"npm", "ci", "--ignore-scripts"},
			want: []string{"npm", "ci", "--ignore-scripts"},
		},
		{
			name: "never applies to npm add and i",
			p:    policy.Policy{NpmWrapScripts: policy.NpmWrapNever},
			args: []string{"npm", "add", "foo"},
			want: []string{"npm", "add", "foo", "--ignore-scripts"},
		},
		{
			name: "never skips non-install commands",
			p:    policy.Policy{NpmWrapScripts: policy.NpmWrapNever},
			args: []string{"npm", "run", "build"},
			want: []string{"npm", "run", "build"},
		},
		{
			name: "never skips pip",
			p:    policy.Policy{NpmWrapScripts: policy.NpmWrapNever},
			args: []string{"pip", "install", "requests"},
			want: []string{"pip", "install", "requests"},
		},
		{
			name: "allowlisted empty forces ignore-scripts",
			p:    policy.Policy{NpmWrapScripts: policy.NpmWrapAllowlisted},
			args: []string{"npm", "install"},
			want: []string{"npm", "install", "--ignore-scripts"},
		},
		{
			name: "allowlisted non-empty keeps sandboxed argv",
			p: policy.Policy{
				NpmWrapScripts:     policy.NpmWrapAllowlisted,
				NpmScriptAllowlist: []string{"esbuild"},
			},
			args: []string{"npm", "install"},
			want: []string{"npm", "install"},
		},
		{
			name: "empty mode treated as always",
			p:    policy.Policy{},
			args: []string{"npm", "install"},
			want: []string{"npm", "install"},
		},
		{
			name: "handles prefixed npm path and flags before subcommand",
			p:    policy.Policy{NpmWrapScripts: policy.NpmWrapNever},
			args: []string{"/usr/bin/npm", "--prefix", "/app", "i"},
			want: []string{"/usr/bin/npm", "--prefix", "/app", "i", "--ignore-scripts"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := policy.ApplyNpmWrapScripts(tt.p, tt.args)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v want %v", got, tt.want)
				}
			}
		})
	}
}
