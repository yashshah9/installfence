package violations

import (
	"bytes"
	"strings"
	"testing"
)

func TestCollectorCapsAtThousand(t *testing.T) {
	c := Collector{}
	for i := 0; i < 1005; i++ {
		c.Add(Event{Kind: "hide_path", Path: "/tmp"})
	}
	if len(c.Events) != 1000 {
		t.Fatalf("expected 1000 events, got %d", len(c.Events))
	}
}

func TestEmpty(t *testing.T) {
	c := Collector{}
	if !c.Empty() {
		t.Fatal("new collector should be empty")
	}
}

func TestWriteJSONIsNDJSON(t *testing.T) {
	c := Collector{}
	c.Add(Event{Kind: "hide_path", Path: "/tmp/secret", Message: "blocked"})
	c.Add(Event{Kind: "sandbox_exit", Message: "exited 1"})
	var buf bytes.Buffer
	if err := c.WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 NDJSON lines, got %q", buf.String())
	}
	if !strings.Contains(lines[0], `"kind":"hide_path"`) {
		t.Fatalf("unexpected first line: %s", lines[0])
	}
}

func TestParseStderrDenyPatterns(t *testing.T) {
	const fixture = `
normal output that should be ignored
cat: /home/user/.ssh/id_rsa: Permission denied
open(/etc/shadow): Operation not permitted
bwrap: Can't mkdir parents for /secret: Read-only file system
write failed: EACCES
info: all good
`
	c := Collector{}
	c.ParseStderr(fixture)
	if len(c.Events) != 4 {
		t.Fatalf("expected 4 deny events, got %d: %+v", len(c.Events), c.Events)
	}
	if c.Events[0].Kind != "denied" || c.Events[0].Path != "/home/user/.ssh/id_rsa" {
		t.Fatalf("unexpected first event: %+v", c.Events[0])
	}
	if !strings.Contains(c.Events[1].Message, "Operation not permitted") {
		t.Fatalf("unexpected second event: %+v", c.Events[1])
	}
	if !strings.HasPrefix(c.Events[2].Message, "bwrap:") {
		t.Fatalf("expected bwrap line, got %+v", c.Events[2])
	}
	if !strings.Contains(c.Events[3].Message, "EACCES") {
		t.Fatalf("unexpected fourth event: %+v", c.Events[3])
	}
}

func TestParseStderrIgnoresBenign(t *testing.T) {
	c := Collector{}
	c.ParseStderr("installing requests==2.0.0\nDownloading wheel\n")
	if !c.Empty() {
		t.Fatalf("expected no violations, got %+v", c.Events)
	}
}
