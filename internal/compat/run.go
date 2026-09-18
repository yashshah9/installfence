package compat

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/yashshah9/installfence/internal/policy"
	"github.com/yashshah9/installfence/internal/sandbox"
)

// Mode selects dry-run planning vs real sandboxed installs.
type Mode string

const (
	ModeDryRun  Mode = "dry-run"
	ModeInstall Mode = "install"
)

// RunOptions configures a matrix execution.
type RunOptions struct {
	Mode           Mode
	PolicyPath     string
	RequireSandbox bool
	FailOnViolation bool
	WorkRoot       string // parent for per-entry temp dirs; empty → os.TempDir
}

// EntryResult is the outcome for one matrix package.
type EntryResult struct {
	ID           string  `json:"id"`
	Ecosystem    string  `json:"ecosystem"`
	Package      string  `json:"package"`
	Mode         string  `json:"mode"`
	OK           bool    `json:"ok"`
	ExitCode     int     `json:"exit_code"`
	ExpectExit   int     `json:"expect_exit"`
	DurationMS   int64   `json:"duration_ms"`
	Message      string  `json:"message,omitempty"`
	Error        string  `json:"error,omitempty"`
	Violations   int     `json:"violations"`
	Command      string  `json:"command,omitempty"`
}

// Report aggregates matrix results.
type Report struct {
	Total   int           `json:"total"`
	Passed  int           `json:"passed"`
	Failed  int           `json:"failed"`
	PassRate float64      `json:"pass_rate"`
	Results []EntryResult `json:"results"`
}

// Run executes filtered matrix entries.
func Run(entries []Entry, opts RunOptions) (Report, error) {
	if opts.Mode == "" {
		opts.Mode = ModeDryRun
	}
	p, err := policy.Load(opts.PolicyPath)
	if err != nil {
		return Report{}, err
	}
	if opts.Mode == ModeDryRun {
		p.DryRun = true
	}

	root := opts.WorkRoot
	if root == "" {
		root = os.TempDir()
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return Report{}, err
	}

	rep := Report{Results: make([]EntryResult, 0, len(entries))}
	for _, e := range entries {
		rep.Results = append(rep.Results, runOne(e, p, opts, root))
	}
	for _, r := range rep.Results {
		rep.Total++
		if r.OK {
			rep.Passed++
		} else {
			rep.Failed++
		}
	}
	if rep.Total > 0 {
		rep.PassRate = float64(rep.Passed) / float64(rep.Total)
	}
	return rep, nil
}

func runOne(e Entry, base policy.Policy, opts RunOptions, root string) EntryResult {
	start := time.Now()
	res := EntryResult{
		ID:         e.ID,
		Ecosystem:  e.Ecosystem,
		Package:    e.Package,
		Mode:       string(opts.Mode),
		ExpectExit: e.ExpectExit,
	}

	work := filepath.Join(root, "compat-"+e.ID)
	_ = os.RemoveAll(work)
	if err := os.MkdirAll(work, 0o755); err != nil {
		res.Error = err.Error()
		res.DurationMS = time.Since(start).Milliseconds()
		return res
	}
	defer os.RemoveAll(work)

	args, err := InstallArgs(e, work)
	if err != nil {
		res.Error = err.Error()
		res.DurationMS = time.Since(start).Milliseconds()
		return res
	}
	res.Command = joinArgs(args)

	p := base
	if e.PolicyFile != "" {
		loaded, err := policy.Load(e.PolicyFile)
		if err != nil {
			res.Error = err.Error()
			res.DurationMS = time.Since(start).Milliseconds()
			return res
		}
		p = loaded
		if opts.Mode == ModeDryRun {
			p.DryRun = true
		}
	}
	// Writable workdir for nested installs under ro-bind root.
	p.AllowWritePaths = append(append([]string{}, p.AllowWritePaths...), work)

	result, err := sandbox.RunWithOptions(p, args, sandbox.Options{
		RequireSandbox:  opts.RequireSandbox && opts.Mode == ModeInstall,
		FailOnViolation: false, // matrix aggregates; CLI decides fail-under
	})
	res.DurationMS = time.Since(start).Milliseconds()
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.ExitCode = result.ExitCode
	res.Message = result.Message
	if result.Violations != nil {
		res.Violations = len(result.Violations.Events)
	}
	res.OK = result.ExitCode == e.ExpectExit
	if opts.Mode == ModeInstall && opts.FailOnViolation && result.Violations != nil && !result.Violations.Empty() {
		res.OK = false
		if res.Error == "" {
			res.Error = "violations recorded"
		}
	}
	return res
}

func joinArgs(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += a
	}
	return out
}

// WriteJSON writes the report as pretty JSON.
func (r Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteText writes a human summary.
func (r Report) WriteText(w io.Writer) {
	fmt.Fprintf(w, "compat: %d/%d passed (%.0f%%)\n", r.Passed, r.Total, r.PassRate*100)
	for _, e := range r.Results {
		status := "PASS"
		if !e.OK {
			status = "FAIL"
		}
		extra := e.Message
		if e.Error != "" {
			extra = e.Error
		}
		fmt.Fprintf(w, "  %-5s %-28s exit=%d expect=%d %s\n", status, e.ID, e.ExitCode, e.ExpectExit, extra)
	}
}
