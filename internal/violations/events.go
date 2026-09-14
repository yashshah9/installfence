package violations

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const FailExitCode = 42

// Event is a single policy denial or sandbox failure.
type Event struct {
	Kind    string `json:"kind"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

type Collector struct {
	Events []Event
}

func (c *Collector) Add(event Event) {
	if len(c.Events) >= 1000 {
		return
	}
	c.Events = append(c.Events, event)
}

func (c *Collector) Empty() bool {
	return len(c.Events) == 0
}

// ParseStderr scans bwrap/sandbox command stderr for typical deny patterns
// and appends structured violation events.
func (c *Collector) ParseStderr(stderr string) {
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || !isDenyLine(trimmed) {
			continue
		}
		c.Add(Event{
			Kind:    "denied",
			Path:    extractPath(trimmed),
			Message: trimmed,
		})
	}
}

func isDenyLine(line string) bool {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "permission denied"),
		strings.Contains(lower, "operation not permitted"),
		strings.Contains(line, "EACCES"),
		strings.Contains(lower, "bwrap:"):
		return true
	default:
		return false
	}
}

// extractPath returns the first absolute path-looking token, if any.
func extractPath(line string) string {
	for _, field := range strings.FieldsFunc(line, func(r rune) bool {
		return r == ' ' || r == '"' || r == '\'' || r == ':'
	}) {
		if strings.HasPrefix(field, "/") && len(field) > 1 {
			return field
		}
	}
	return ""
}

func (c *Collector) WriteJSON(w io.Writer) error {
	for _, event := range c.Events {
		if err := json.NewEncoder(w).Encode(event); err != nil {
			return err
		}
	}
	return nil
}

func (c *Collector) WriteText(w io.Writer) {
	for _, event := range c.Events {
		fmt.Fprintf(w, "violation: %s %s %s\n", event.Kind, event.Path, event.Message)
	}
}
