package sandbox

import (
	"strings"
	"testing"

	"github.com/installfence/installfence/internal/policy"
)

func TestBuildBwrapArgsUnsetsEnv(t *testing.T) {
	p := policy.Policy{HideEnvKeys: []string{"GITHUB_TOKEN"}, AllowNetwork: true}
	args := buildBwrapArgs(p, []string{"echo", "hi"})
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--unsetenv GITHUB_TOKEN") {
		t.Fatalf("expected --unsetenv GITHUB_TOKEN, got %v", args)
	}
	if !strings.Contains(joined, "-- echo hi") && !(contains(args, "echo") && contains(args, "hi")) {
		t.Fatalf("expected command after --, got %v", args)
	}
}

func TestBuildBwrapArgsUnsharesNetWhenDenied(t *testing.T) {
	p := policy.Policy{AllowNetwork: false}
	args := buildBwrapArgs(p, []string{"true"})
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--unshare-net") {
		t.Fatalf("expected --unshare-net, got %v", args)
	}
}

func TestBuildBwrapArgsAllowWritePaths(t *testing.T) {
	dir := t.TempDir()
	p := policy.Policy{AllowNetwork: true, AllowWritePaths: []string{dir}}
	args := buildBwrapArgs(p, []string{"true"})
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--bind "+dir+" "+dir) {
		t.Fatalf("expected --bind for allow_write_paths, got %v", args)
	}
}

func TestBuildBwrapArgsAllowWritePathsEdges(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("IF_TEST_DIR", dir)

	tests := []struct {
		name     string
		paths    []string
		wantBind bool
		bindPath string
	}{
		{name: "empty path skipped", paths: []string{""}, wantBind: false},
		{name: "missing path skipped", paths: []string{"/nonexistent/installfence-test-path"}, wantBind: false},
		{name: "env expansion", paths: []string{"${IF_TEST_DIR}"}, wantBind: true, bindPath: dir},
		{name: "mixed valid and invalid", paths: []string{"", dir, "/no/such/path"}, wantBind: true, bindPath: dir},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := policy.Policy{AllowNetwork: true, AllowWritePaths: tt.paths}
			args := buildBwrapArgs(p, []string{"true"})
			joined := strings.Join(args, " ")
			hasBind := strings.Contains(joined, "--bind")
			if hasBind != tt.wantBind {
				t.Fatalf("--bind present=%v, want %v; args=%v", hasBind, tt.wantBind, args)
			}
			if tt.wantBind && tt.bindPath != "" && !strings.Contains(joined, "--bind "+tt.bindPath+" "+tt.bindPath) {
				t.Fatalf("expected --bind %s, got %v", tt.bindPath, args)
			}
		})
	}
}

func TestScrubEnvRemovesHiddenKeys(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "secret")
	t.Setenv("KEEP_ME", "yes")
	out := scrubEnv([]string{"GITHUB_TOKEN"})
	for _, kv := range out {
		if strings.HasPrefix(kv, "GITHUB_TOKEN=") {
			t.Fatal("GITHUB_TOKEN should be scrubbed")
		}
	}
}

func TestRequireSandboxErrorsWhenMissing(t *testing.T) {
	if HasBubblewrap() || HasSandboxExec() {
		t.Skip("sandbox backend present")
	}
	_, err := RunWithOptions(policy.Policy{}, []string{"true"}, Options{RequireSandbox: true})
	if err == nil {
		t.Fatal("expected error when sandbox backend is missing")
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
