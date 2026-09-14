package policy

import (
	"path/filepath"
	"strings"
)

const (
	NpmWrapAlways      = "always"
	NpmWrapAllowlisted = "allowlisted"
	NpmWrapNever       = "never"
)

// ApplyNpmWrapScripts mutates npm/npx install-like argv per NpmWrapScripts.
//
// Modes:
//   - always: unchanged (current sandboxed behavior)
//   - never: append --ignore-scripts to install/ci/add if missing
//   - allowlisted: empty allowlist → same as never; non-empty → unchanged
//     (allowlist reserved for future script-name filtering)
func ApplyNpmWrapScripts(p Policy, args []string) []string {
	if len(args) == 0 || !isNpmOrNpx(args[0]) || !isNpmInstallLike(args) {
		return args
	}
	mode := p.NpmWrapScripts
	if mode == "" {
		mode = NpmWrapAlways
	}
	forceIgnore := false
	switch mode {
	case NpmWrapNever:
		forceIgnore = true
	case NpmWrapAllowlisted:
		forceIgnore = len(p.NpmScriptAllowlist) == 0
	default:
		return args
	}
	if !forceIgnore || hasIgnoreScripts(args) {
		return args
	}
	out := make([]string, len(args), len(args)+1)
	copy(out, args)
	return append(out, "--ignore-scripts")
}

func isNpmOrNpx(bin string) bool {
	base := filepath.Base(bin)
	return base == "npm" || base == "npx"
}

// isNpmInstallLike reports whether argv is an npm/npx install/ci/add-style command.
func isNpmInstallLike(args []string) bool {
	for i := 1; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			continue
		}
		if strings.HasPrefix(a, "-") {
			key, _, hasEq := strings.Cut(a, "=")
			if !hasEq && npmFlagTakesValue(key) && i+1 < len(args) {
				i++
			}
			continue
		}
		switch a {
		case "install", "i", "ci", "add":
			return true
		default:
			return false
		}
	}
	return false
}

func npmFlagTakesValue(flag string) bool {
	switch flag {
	case "--prefix", "-C", "--workspace", "-w", "--userconfig", "--globalconfig", "--cache":
		return true
	default:
		return false
	}
}

func hasIgnoreScripts(args []string) bool {
	for _, a := range args {
		if a == "--ignore-scripts" {
			return true
		}
	}
	return false
}
