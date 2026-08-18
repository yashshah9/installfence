package violations

import (
	"encoding/json"
	"fmt"
	"io"
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
