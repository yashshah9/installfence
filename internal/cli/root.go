package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/installfence/installfence/internal/policy"
	"github.com/installfence/installfence/internal/sandbox"
	"github.com/installfence/installfence/internal/violations"
	"github.com/spf13/cobra"
)

const Version = "0.4.0"

var (
	policyFile        string
	dryRun            bool
	requireSandbox    bool
	failOnViolation   bool
	jsonViolations    bool
)

func Execute() error {
	return newRoot().Execute()
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "installfence",
		Short:         "Sandboxed package installation for pip, uv, npm, and more",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
	}

	root.PersistentFlags().StringVar(&policyFile, "policy", "", "Path to policy YAML")
	root.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Print sandbox plan without enforcing")
	root.PersistentFlags().BoolVar(&requireSandbox, "require-sandbox", false, "Fail if no sandbox backend is available")
	root.PersistentFlags().BoolVar(&failOnViolation, "fail-on-violation", false, "Exit 42 when a violation is recorded")
	root.PersistentFlags().BoolVar(&jsonViolations, "json-violations", false, "Emit violations as NDJSON on stderr")

	root.AddCommand(healthCmd())
	root.AddCommand(runCmd())
	root.AddCommand(shimCmd())
	root.AddCommand(wrapCmd("pip"))
	root.AddCommand(wrapCmd("uv"))
	root.AddCommand(wrapCmd("npm"))

	return root
}

func healthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check installfence installation and sandbox availability",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("installfence %s OK\n", Version)
			if sandbox.HasBubblewrap() {
				fmt.Println("  bubblewrap: available")
			} else {
				fmt.Println("  bubblewrap: NOT FOUND")
			}
			if sandbox.HasSandboxExec() {
				fmt.Println("  sandbox-exec: available")
			}
			fmt.Println("  flags like --dry-run and --require-sandbox go before the tool name:")
			fmt.Println("    installfence --dry-run pip install requests")
			return nil
		},
	}
}

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run [command...]",
		Short: "Run an arbitrary command inside the sandbox",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeSandboxed(args)
		},
	}
}

func wrapCmd(tool string) *cobra.Command {
	return &cobra.Command{
		Use:                tool,
		Short:              fmt.Sprintf("Run %s inside the sandbox", tool),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeSandboxed(append([]string{tool}, args...))
		},
	}
}

func executeSandboxed(args []string) error {
	p, err := policy.Load(policyFile)
	if err != nil {
		return err
	}
	if dryRun {
		p.DryRun = true
	}
	result, err := sandbox.RunWithOptions(p, args, sandbox.Options{
		RequireSandbox:  requireSandbox,
		FailOnViolation: failOnViolation,
	})
	if err != nil {
		if isNamespaceError(err) {
			return fmt.Errorf("%w\nDocker Compose needs privileged: true for bubblewrap user namespaces", err)
		}
		return err
	}
	if result.Violations != nil && !result.Violations.Empty() {
		if jsonViolations {
			_ = result.Violations.WriteJSON(os.Stderr)
		} else {
			result.Violations.WriteText(os.Stderr)
		}
	}
	fmt.Fprintf(os.Stderr, "installfence: %s (exit %d)\n", result.Message, result.ExitCode)
	if failOnViolation && result.Violations != nil && !result.Violations.Empty() {
		os.Exit(violations.FailExitCode)
	}
	if result.ExitCode != 0 {
		os.Exit(result.ExitCode)
	}
	return nil
}

func isNamespaceError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "namespace") || strings.Contains(msg, "operation not permitted")
}
