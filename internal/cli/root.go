package cli

import (
	"fmt"
	"os"

	"github.com/installfence/installfence/internal/policy"
	"github.com/installfence/installfence/internal/sandbox"
	"github.com/spf13/cobra"
)

var (
	policyFile string
	dryRun     bool
)

// Execute runs the root command.
func Execute() error {
	return newRoot().Execute()
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "installfence",
		Short: "Sandboxed package installation for pip, uv, npm, and more",
	}

	root.PersistentFlags().StringVar(&policyFile, "policy", "", "Path to policy YAML")
	root.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Print sandbox plan without enforcing")

	root.AddCommand(healthCmd())
	root.AddCommand(runCmd())
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
			fmt.Println("installfence OK")
			if sandbox.HasBubblewrap() {
				fmt.Println("  bubblewrap: available")
			} else {
				fmt.Println("  bubblewrap: NOT FOUND (install bubblewrap for sandboxing)")
			}
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
		Use:            tool,
		Short:          fmt.Sprintf("Run %s inside the sandbox", tool),
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
	result, err := sandbox.Run(p, args)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "installfence: %s (exit %d)\n", result.Message, result.ExitCode)
	if result.ExitCode != 0 {
		os.Exit(result.ExitCode)
	}
	return nil
}
