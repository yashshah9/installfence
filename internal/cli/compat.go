package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yashshah9/installfence/internal/compat"
)

func compatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compat",
		Short: "Run the package compatibility matrix under the sandbox",
	}
	cmd.AddCommand(compatListCmd())
	cmd.AddCommand(compatRunCmd())
	return cmd
}

func defaultMatrixPath() string {
	if p := os.Getenv("INSTALLFENCE_COMPAT_MATRIX"); p != "" {
		return p
	}
	candidates := []string{
		"config/compat-matrix.yaml",
		"../config/compat-matrix.yaml",
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "compat-matrix.yaml"))
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "config/compat-matrix.yaml"
}

func compatListCmd() *cobra.Command {
	var (
		matrix string
		tag    []string
		eco    []string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List matrix packages (optionally filtered)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if matrix == "" {
				matrix = defaultMatrixPath()
			}
			m, err := compat.Load(matrix)
			if err != nil {
				return err
			}
			entries := m.Filter(eco, tag, 0)
			fmt.Printf("%d package(s) from %s\n", len(entries), matrix)
			for _, e := range entries {
				ver := e.Version
				if ver == "" {
					ver = "*"
				}
				fmt.Printf("  %-28s %-4s %s@%s tags=%v\n", e.ID, e.Ecosystem, e.Package, ver, e.Tags)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&matrix, "matrix", "", "Path to compat matrix YAML")
	cmd.Flags().StringSliceVar(&tag, "tag", nil, "Only entries with these tags (e.g. smoke)")
	cmd.Flags().StringSliceVar(&eco, "ecosystem", nil, "Only these ecosystems (pip,npm,uv)")
	return cmd
}

func compatRunCmd() *cobra.Command {
	var (
		matrix     string
		tag        []string
		eco        []string
		limit      int
		mode       string
		jsonOut    bool
		failUnder  float64
		workRoot   string
	)
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute matrix entries (dry-run or real sandboxed installs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if matrix == "" {
				matrix = defaultMatrixPath()
			}
			m, err := compat.Load(matrix)
			if err != nil {
				return err
			}
			entries := m.Filter(eco, tag, limit)
			if len(entries) == 0 {
				return fmt.Errorf("no matrix entries matched filters")
			}
			runMode := compat.ModeDryRun
			switch mode {
			case "dry-run", "":
				runMode = compat.ModeDryRun
			case "install":
				runMode = compat.ModeInstall
			default:
				return fmt.Errorf("unknown mode %q (want dry-run|install)", mode)
			}

			rep, err := compat.Run(entries, compat.RunOptions{
				Mode:            runMode,
				PolicyPath:      policyFile,
				RequireSandbox:  requireSandbox,
				FailOnViolation: failOnViolation,
				WorkRoot:        workRoot,
			})
			if err != nil {
				return err
			}
			if jsonOut {
				if err := rep.WriteJSON(os.Stdout); err != nil {
					return err
				}
			} else {
				rep.WriteText(os.Stdout)
			}
			if rep.PassRate < failUnder {
				return fmt.Errorf("compat pass rate %.2f < fail-under %.2f", rep.PassRate, failUnder)
			}
			if rep.Failed > 0 && failUnder >= 1.0 {
				return fmt.Errorf("%d compat failure(s)", rep.Failed)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&matrix, "matrix", "", "Path to compat matrix YAML")
	cmd.Flags().StringSliceVar(&tag, "tag", nil, "Only entries with these tags")
	cmd.Flags().StringSliceVar(&eco, "ecosystem", nil, "Only these ecosystems")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max entries to run (0 = all matched)")
	cmd.Flags().StringVar(&mode, "mode", "dry-run", "dry-run | install")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit JSON report on stdout")
	cmd.Flags().Float64Var(&failUnder, "fail-under", 1.0, "Fail if pass rate is below this (0.0–1.0)")
	cmd.Flags().StringVar(&workRoot, "work-root", "", "Parent directory for install workdirs")
	return cmd
}
