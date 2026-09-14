package cli

import (
	"fmt"
	"os"
	"runtime"

	"github.com/yashshah9/installfence/internal/shim"
	"github.com/spf13/cobra"
)

func shimCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shim",
		Short: "Install PATH shims so pip/npm/uv run inside installfence",
	}
	var dir string
	var policy string
	var shell string
	cmd.PersistentFlags().StringVar(&dir, "dir", "", "Shim directory (default ~/.installfence/shims)")
	cmd.PersistentFlags().StringVar(&policy, "shim-policy", "", "Policy YAML passed to shimmed commands")
	cmd.PersistentFlags().StringVar(&shell, "shell", defaultShell(), "Shell for env snippet: bash, zsh, fish")

	cmd.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Write shim scripts for pip, uv, npm, pnpm, and yarn",
		RunE: func(c *cobra.Command, args []string) error {
			target := shimDir(dir)
			if err := shim.Install(target, "", policy); err != nil {
				return err
			}
			fmt.Printf("installed shims in %s\n", target)
			fmt.Printf("add to your shell: eval \"$(installfence shim env --dir %q)\"\n", target)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "uninstall",
		Short: "Remove shim scripts",
		RunE: func(c *cobra.Command, args []string) error {
			target := shimDir(dir)
			if err := shim.Uninstall(target); err != nil {
				return err
			}
			fmt.Printf("removed shims from %s\n", target)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show whether shims are on PATH",
		RunE: func(c *cobra.Command, args []string) error {
			target := shimDir(dir)
			if _, err := os.Stat(target); os.IsNotExist(err) {
				fmt.Printf("shims not installed (run: installfence shim install --dir %q)\n", target)
				return nil
			}
			for _, info := range shim.Status(target, os.Getenv("PATH")) {
				flag := "not on PATH"
				if info.OnPATH && info.BeforeOS {
					flag = "active (first on PATH)"
				} else if info.OnPATH {
					flag = "on PATH (not first)"
				}
				fmt.Printf("  %-6s %s\n", info.Name, flag)
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "env",
		Short: "Print a shell snippet that prepends the shim directory to PATH",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Print(shim.EnvSnippet(shimDir(dir), shell))
			return nil
		},
	})
	return cmd
}

func shimDir(dir string) string {
	if dir != "" {
		return dir
	}
	return shim.DefaultDir()
}

func defaultShell() string {
	if runtime.GOOS == "windows" {
		return "bash"
	}
	return "zsh"
}
