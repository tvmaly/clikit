package worker

import (
	"encoding/json"
	"io"
	"time"
)

// Event is the structured worker event contract. It intentionally excludes
// payload bodies, bearer tokens, and environment values.
type Event struct {
	Name       string `json:"event"`
	RequestID  string `json:"request_id,omitempty"`
	Tool       string `json:"tool,omitempty"`
	Operation  string `json:"operation,omitempty"`
	WorkerID   string `json:"worker_id,omitempty"`
	DurationMS int64  `json:"duration_ms,omitempty"`
	State      string `json:"state,omitempty"`
	Retry      bool   `json:"retry,omitempty"`
	ErrorClass string `json:"error_class,omitempty"`
}

// EventHook receives structured worker events.
type EventHook func(Event)

// WriteEventJSON writes one event as compact JSON.
func WriteEventJSON(w io.Writer, e Event) error {
	return json.NewEncoder(w).Encode(e)
}

func (d *Dispatcher) emit(e Event) {
	if e.WorkerID == "" {
		e.WorkerID = d.WorkerID
	}
	if d.EventHook != nil {
		d.EventHook(e)
	}
}

func durationMS(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
