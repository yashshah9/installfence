package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var Tools = []string{"pip", "pip3", "uv", "npm", "pnpm", "yarn"}

type Info struct {
	Name     string
	Path     string
	OnPATH   bool
	BeforeOS bool
}

func DefaultDir() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = os.TempDir()
	}
	return filepath.Join(home, ".installfence", "shims")
}

func Install(dir, binary, policy string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if binary == "" {
		var err error
		binary, err = os.Executable()
		if err != nil {
			return err
		}
	}
	for _, tool := range Tools {
		body := script(binary, policy, tool)
		path := filepath.Join(dir, tool)
		if runtime.GOOS == "windows" {
			path += ".cmd"
		}
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func Uninstall(dir string) error {
	for _, tool := range Tools {
		_ = os.Remove(filepath.Join(dir, tool))
		_ = os.Remove(filepath.Join(dir, tool+".cmd"))
	}
	return nil
}

func Status(dir, pathEnv string) []Info {
	entries := filepath.SplitList(pathEnv)
	var out []Info
	for _, tool := range Tools {
		shimPath := filepath.Join(dir, tool)
		info := Info{Name: tool, Path: shimPath}
		for i, entry := range entries {
			if filepath.Clean(entry) == filepath.Clean(dir) {
				info.OnPATH = true
				info.BeforeOS = i == 0
				break
			}
		}
		out = append(out, info)
	}
	return out
}

func EnvSnippet(dir, shell string) string {
	switch shell {
	case "fish":
		return fmt.Sprintf("set -gx PATH %s $PATH\n", dir)
	default:
		return fmt.Sprintf("export PATH=%q:$PATH\n", dir)
	}
}

func script(binary, policy, tool string) string {
	policyFlag := ""
	if policy != "" {
		policyFlag = fmt.Sprintf(" --policy %s", quote(policy))
	}
	return fmt.Sprintf("#!/bin/sh\nexec %s%s run -- %s \"$@\"\n", quote(binary), policyFlag, tool)
}

func quote(path string) string {
	if path == "" {
		return path
	}
	if strings.ContainsAny(path, " \t\"'") {
		return "'" + strings.ReplaceAll(path, "'", "'\\''") + "'"
	}
	return path
}
