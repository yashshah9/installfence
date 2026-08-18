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
