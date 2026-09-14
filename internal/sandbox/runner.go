package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/installfence/installfence/internal/policy"
	"github.com/installfence/installfence/internal/violations"
)

// Result captures sandbox execution outcome.
type Result struct {
	Command    string
	ExitCode   int
	DryRun     bool
	Message    string
	Violations *violations.Collector
}

func HasBubblewrap() bool {
	_, err := exec.LookPath("bwrap")
	return err == nil
}

func HasSandboxExec() bool {
	_, err := exec.LookPath("sandbox-exec")
	return err == nil
}

type Options struct {
	RequireSandbox bool
	FailOnViolation bool
}

func Run(p policy.Policy, args []string) (Result, error) {
	return RunWithOptions(p, args, Options{})
}

func RunWithOptions(p policy.Policy, args []string, opts Options) (Result, error) {
	if len(args) == 0 {
		return Result{}, fmt.Errorf("no command provided")
	}
	cmdStr := strings.Join(args, " ")
	collector := &violations.Collector{}

	if p.DryRun {
		return Result{Command: cmdStr, ExitCode: 0, DryRun: true, Message: "dry-run: would sandbox command", Violations: collector}, nil
	}

	if runtime.GOOS == "linux" && HasBubblewrap() {
		return runBwrap(p, args, collector)
	}
	if runtime.GOOS == "darwin" && HasSandboxExec() {
		return runSandboxExec(p, args, collector)
	}

	if opts.RequireSandbox {
		return Result{}, fmt.Errorf("sandbox backend not available (need bubblewrap on Linux or sandbox-exec on macOS)")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = scrubEnv(p.HideEnvKeys)
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return Result{}, err
		}
	}
	return Result{
		Command:    cmdStr,
		ExitCode:   exitCode,
		DryRun:     false,
		Message:    "sandbox backend not found; running passthrough (unsafe)",
		Violations: collector,
	}, nil
}

func runBwrap(p policy.Policy, args []string, collector *violations.Collector) (Result, error) {
	bwrapArgs := buildBwrapArgs(p, args)
	cmd := exec.Command("bwrap", bwrapArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = scrubEnv(p.HideEnvKeys)
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			collector.Add(violations.Event{
				Kind:    "sandbox_exit",
				Message: fmt.Sprintf("sandboxed process exited %d", exitCode),
			})
		} else {
			return Result{}, fmt.Errorf("sandbox exec: %w", err)
		}
	}
	return Result{
		Command:    strings.Join(args, " "),
		ExitCode:   exitCode,
		Message:    "sandboxed with bubblewrap",
		Violations: collector,
	}, nil
}

func runSandboxExec(p policy.Policy, args []string, collector *violations.Collector) (Result, error) {
	profile := darwinProfile(p)
	tmp, err := os.CreateTemp("", "installfence-*.sb")
	if err != nil {
		return Result{}, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(profile); err != nil {
		return Result{}, err
	}
	tmp.Close()

	cmdArgs := append([]string{"-f", tmp.Name()}, args...)
	cmd := exec.Command("sandbox-exec", cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = scrubEnv(p.HideEnvKeys)
	runErr := cmd.Run()
	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			collector.Add(violations.Event{Kind: "sandbox_exit", Message: runErr.Error()})
		} else {
			return Result{}, runErr
		}
	}
	return Result{
		Command:    strings.Join(args, " "),
		ExitCode:   exitCode,
		Message:    "sandboxed with sandbox-exec",
		Violations: collector,
	}, nil
}

func darwinProfile(p policy.Policy) string {
	b := strings.Builder{}
	b.WriteString("(version 1)\n(deny default)\n(allow process-exec)\n(allow process-fork)\n(allow file-read*)\n(allow file-write*)\n(allow sysctl-read)\n(allow mach-lookup)\n(allow network-outbound)\n")
	if !p.AllowNetwork {
		b.WriteString("(deny network*)\n")
	}
	for _, path := range p.HidePaths {
		if path == "" {
			continue
		}
		fmt.Fprintf(&b, "(deny file-read* (regex #\"%s\"))\n", path)
	}
	return b.String()
}

func scrubEnv(keys []string) []string {
	deny := map[string]struct{}{}
	for _, k := range keys {
		deny[k] = struct{}{}
	}
	var out []string
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if _, hidden := deny[name]; hidden {
			continue
		}
		out = append(out, kv)
	}
	return out
}

func buildBwrapArgs(p policy.Policy, args []string) []string {
	bwrap := []string{
		"--unshare-all",
		"--share-net",
		"--die-with-parent",
		"--ro-bind", "/", "/",
		"--dev", "/dev",
		"--proc", "/proc",
		"--tmpfs", "/tmp",
	}
	for _, path := range p.HidePaths {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			bwrap = append(bwrap, "--tmpfs", path)
		}
	}
	for _, path := range p.AllowWritePaths {
		if path == "" {
			continue
		}
		expanded := os.ExpandEnv(path)
		if _, err := os.Stat(expanded); err == nil {
			bwrap = append(bwrap, "--bind", expanded, expanded)
		}
	}
	for _, key := range p.HideEnvKeys {
		bwrap = append(bwrap, "--unsetenv", key)
	}
	if !p.AllowNetwork {
		bwrap = append(bwrap, "--unshare-net")
	}
	bwrap = append(bwrap, "--")
	bwrap = append(bwrap, args...)
	return bwrap
}
