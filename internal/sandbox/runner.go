package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/installfence/installfence/internal/policy"
)

// Result captures sandbox execution outcome.
type Result struct {
	Command string
	ExitCode int
	DryRun   bool
	Message  string
}

// HasBubblewrap reports whether bwrap is available on PATH.
func HasBubblewrap() bool {
	_, err := exec.LookPath("bwrap")
	return err == nil
}

// Run executes args under bubblewrap with the given policy, or passthrough in dry-run.
func Run(policy policy.Policy, args []string) (Result, error) {
	if len(args) == 0 {
		return Result{}, fmt.Errorf("no command provided")
	}

	cmdStr := strings.Join(args, " ")
	if policy.DryRun || !HasBubblewrap() {
		msg := "dry-run: would sandbox command"
		if !HasBubblewrap() {
			msg = "bwrap not found; running passthrough (unsafe)"
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return Result{}, err
			}
		}
		return Result{Command: cmdStr, ExitCode: exitCode, DryRun: policy.DryRun || !HasBubblewrap(), Message: msg}, nil
	}

	bwrapArgs := buildBwrapArgs(policy, args)
	cmd := exec.Command("bwrap", bwrapArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return Result{}, fmt.Errorf("sandbox exec: %w", err)
		}
	}
	return Result{Command: cmdStr, ExitCode: exitCode, Message: "sandboxed with bubblewrap"}, nil
}

func buildBwrapArgs(p policy.Policy, args []string) []string {
	bwrap := []string{
		"--unshare-all",
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
		// Hide by binding an empty tmpfs over the path if it exists
		if _, err := os.Stat(path); err == nil {
			bwrap = append(bwrap, "--tmpfs", path)
		}
	}

	if !p.AllowNetwork {
		bwrap = append(bwrap, "--unshare-net")
	}

	bwrap = append(bwrap, "--")
	bwrap = append(bwrap, args...)
	return bwrap
}
