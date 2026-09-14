package violations

import (
	"strings"
	"testing"
)

func TestParseStderrScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		stderr     string
		wantCount  int
		wantPaths  []string
		wantSubstr []string // each event message must contain one of these (indexed by event)
	}{
		{
			name:      "empty input",
			stderr:    "",
			wantCount: 0,
		},
		{
			name:      "whitespace only",
			stderr:    "   \n\t\n  \r\n",
			wantCount: 0,
		},
		{
			name:      "benign pip install output",
			stderr:    "Collecting requests\nDownloading requests-2.31.0.whl\nSuccessfully installed requests-2.31.0\n",
			wantCount: 0,
		},
		{
			name:      "benign npm output",
			stderr:    "npm WARN deprecated lodash@4.17.20\nadded 42 packages in 3s\n",
			wantCount: 0,
		},
		{
			name:       "permission denied classic",
			stderr:     "cat: /home/user/.ssh/id_rsa: Permission denied\n",
			wantCount:  1,
			wantPaths:  []string{"/home/user/.ssh/id_rsa"},
			wantSubstr: []string{"Permission denied"},
		},
		{
			name:       "permission denied uppercase",
			stderr:     "cat: /etc/shadow: PERMISSION DENIED\n",
			wantCount:  1,
			wantPaths:  []string{"/etc/shadow"},
			wantSubstr: []string{"PERMISSION DENIED"},
		},
		{
			name:       "permission denied mixed case",
			stderr:     "open /var/secret: Permission Denied\n",
			wantCount:  1,
			wantPaths:  []string{"/var/secret"},
			wantSubstr: []string{"Permission Denied"},
		},
		{
			name:       "operation not permitted",
			stderr:     "open(/etc/shadow): Operation not permitted\n",
			wantCount:  1,
			wantPaths:  []string{""}, // extractPath splits on ':' so open(...) path is not captured
			wantSubstr: []string{"Operation not permitted"},
		},
		{
			name:       "operation not permitted uppercase",
			stderr:     "mkdir /root: OPERATION NOT PERMITTED\n",
			wantCount:  1,
			wantPaths:  []string{"/root"},
			wantSubstr: []string{"OPERATION NOT PERMITTED"},
		},
		{
			name:       "EACCES errno",
			stderr:     "write failed: EACCES\n",
			wantCount:  1,
			wantPaths:  []string{},
			wantSubstr: []string{"EACCES"},
		},
		{
			name:       "EACCES with path",
			stderr:     "open /protected/file: EACCES (Permission denied)\n",
			wantCount:  1,
			wantPaths:  []string{"/protected/file"},
			wantSubstr: []string{"EACCES"},
		},
		{
			name:       "bwrap read-only filesystem",
			stderr:     "bwrap: Can't mkdir parents for /secret: Read-only file system\n",
			wantCount:  1,
			wantPaths:  []string{"/secret"},
			wantSubstr: []string{"bwrap:"},
		},
		{
			name:       "bwrap bind mount failure",
			stderr:     "bwrap: Can't bind mount /home/user/project on /home/user/project: No such file or directory\n",
			wantCount:  1,
			wantPaths:  []string{"/home/user/project"},
			wantSubstr: []string{"bwrap:"},
		},
		{
			name:       "bwrap uppercase prefix still matches",
			stderr:     "BWRAP: sandbox helper error\n",
			wantCount:  1,
			wantPaths:  []string{},
			wantSubstr: []string{"BWRAP:"},
		},
		{
			name: "multiple denials in one blob",
			stderr: `cat: /home/user/.ssh/id_rsa: Permission denied
open(/etc/shadow): Operation not permitted
bwrap: Can't mkdir parents for /secret: Read-only file system
write failed: EACCES
`,
			wantCount: 4,
			wantPaths: []string{"/home/user/.ssh/id_rsa", "", "/secret", ""},
			wantSubstr: []string{
				"Permission denied",
				"Operation not permitted",
				"bwrap:",
				"EACCES",
			},
		},
		{
			name: "mixed benign and deny lines",
			stderr: `Collecting package
Downloading wheel
cat: /tmp/forbidden: Permission denied
Successfully installed foo-1.0.0
npm notice created a lockfile
open(/etc/passwd): Operation not permitted
All done!
`,
			wantCount: 2,
			wantPaths: []string{"/tmp/forbidden", ""},
			wantSubstr: []string{
				"Permission denied",
				"Operation not permitted",
			},
		},
		{
			name: "deny sandwiched between benign stdout-like lines",
			stderr: `Looking in indexes: https://pypi.org/simple
WARNING: Running pip as the 'root' user
touch: cannot touch '/root/.bashrc': Permission denied
WARNING: You are using pip version 21.0
`,
			wantCount:  1,
			wantPaths:  []string{"/root/.bashrc"},
			wantSubstr: []string{"Permission denied"},
		},
		{
			name:      "CRLF line endings",
			stderr:    "cat: /data/secret: Permission denied\r\ninfo: ok\r\n",
			wantCount: 1,
			wantPaths: []string{"/data/secret"},
			wantSubstr: []string{
				"Permission denied",
			},
		},
		{
			name: "duplicate deny on same path",
			stderr: `cat: /home/user/.aws/credentials: Permission denied
cat: /home/user/.aws/credentials: Permission denied
`,
			wantCount: 2,
			wantPaths: []string{"/home/user/.aws/credentials", "/home/user/.aws/credentials"},
			wantSubstr: []string{
				"Permission denied",
				"Permission denied",
			},
		},
		{
			name: "chmod and mkdir deny variants",
			stderr: `chmod: changing permissions of '/opt/app': Operation not permitted
mkdir: cannot create directory '/sys/fs': Permission denied
`,
			wantCount: 2,
			wantPaths: []string{"/opt/app", "/sys/fs"},
			wantSubstr: []string{
				"Operation not permitted",
				"Permission denied",
			},
		},
		{
			name:       "deny line without absolute path",
			stderr:     "fatal: Permission denied\n",
			wantCount:  1,
			wantPaths:  []string{""},
			wantSubstr: []string{"Permission denied"},
		},
		{
			name: "three consecutive bwrap errors",
			stderr: `bwrap: Can't mkdir /a: Read-only file system
bwrap: Can't mkdir /b: Read-only file system
bwrap: Can't mkdir /c: Read-only file system
`,
			wantCount: 3,
			wantPaths: []string{"/a", "/b", "/c"},
			wantSubstr: []string{
				"bwrap:",
				"bwrap:",
				"bwrap:",
			},
		},
		{
			name: "info lines containing denied substring but not deny patterns",
			stderr: `info: access denied by policy engine (not a syscall deny)
debug: retrying after timeout
`,
			wantCount: 0,
		},
		{
			name: "leading and trailing blank lines around denies",
			stderr: `

cat: /var/log/secure: Permission denied

`,
			wantCount:  1,
			wantPaths:  []string{"/var/log/secure"},
			wantSubstr: []string{"Permission denied"},
		},
		{
			name: "realistic bwrap sandbox stderr excerpt",
			stderr: `bwrap: Creating new namespace failed, because unprivileged user namespaces are not allowed
bwrap: Can't bind mount /proc/self/fd on /proc/self/fd: Operation not permitted
installing setuptools
`,
			wantCount: 2,
			wantPaths: []string{"", "/proc/self/fd"},
			wantSubstr: []string{
				"bwrap:",
				"bwrap:",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := Collector{}
			c.ParseStderr(tt.stderr)

			if got := len(c.Events); got != tt.wantCount {
				t.Fatalf("event count: got %d, want %d; events=%+v", got, tt.wantCount, c.Events)
			}
			for i, ev := range c.Events {
				if ev.Kind != "denied" {
					t.Fatalf("event[%d].Kind = %q, want denied", i, ev.Kind)
				}
				if len(tt.wantPaths) > i && ev.Path != tt.wantPaths[i] {
					t.Fatalf("event[%d].Path = %q, want %q (msg=%q)", i, ev.Path, tt.wantPaths[i], ev.Message)
				}
				if len(tt.wantSubstr) > i && !strings.Contains(ev.Message, tt.wantSubstr[i]) {
					t.Fatalf("event[%d].Message = %q, want substring %q", i, ev.Message, tt.wantSubstr[i])
				}
			}
		})
	}
}

func TestParseStderrPreservesOrder(t *testing.T) {
	stderr := "first: /a: Permission denied\nsecond: /b: Operation not permitted\nthird: EACCES\n"
	c := Collector{}
	c.ParseStderr(stderr)
	if len(c.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(c.Events))
	}
	if !strings.Contains(c.Events[0].Message, "first") {
		t.Fatalf("order broken: %+v", c.Events)
	}
	if !strings.Contains(c.Events[1].Message, "second") {
		t.Fatalf("order broken: %+v", c.Events)
	}
	if !strings.Contains(c.Events[2].Message, "third") {
		t.Fatalf("order broken: %+v", c.Events)
	}
}

func TestParseStderrDoesNotMutatePriorEvents(t *testing.T) {
	c := Collector{}
	c.Add(Event{Kind: "hide_path", Path: "/tmp/secret", Message: "blocked"})
	c.ParseStderr("cat: /etc/shadow: Permission denied\n")
	if len(c.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(c.Events))
	}
	if c.Events[0].Kind != "hide_path" {
		t.Fatalf("prior event overwritten: %+v", c.Events[0])
	}
	if c.Events[1].Kind != "denied" {
		t.Fatalf("new event wrong kind: %+v", c.Events[1])
	}
}
