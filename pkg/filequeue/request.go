package filequeue

import "time"

// RequestMeta is the metadata written to pending/<requestID>.meta.json.
type RequestMeta struct {
	RequestID           string            `json:"request_id"`
	Tool                string            `json:"tool"`
	Operation           string            `json:"operation"`
	Method              string            `json:"method"`
	Path                string            `json:"path"`
	Params              map[string]string `json:"params"`
	HasPayload          bool              `json:"has_payload"`
	PayloadPath         string            `json:"payload_path"`
	ResponseMetaPath    string            `json:"response_meta_path"`
	ResponsePayloadPath string            `json:"response_payload_path"`
	TimeoutSeconds      int               `json:"timeout_seconds"`
	CreatedAt           time.Time         `json:"created_at"`
	ExpectedResponse    string            `json:"expected_response"`
}

// ResponseMeta is the metadata written to done/<requestID>.meta.json
// (or dead/<requestID>.meta.json for dead-letter requests).
type ResponseMeta struct {
	RequestID   string    `json:"request_id"`
	Status      string    `json:"status"`       // "success" or "error"
	StatusCode  int       `json:"status_code"`
	HasPayload  bool      `json:"has_payload"`
	PayloadPath string    `json:"payload_path"`
	Error       string    `json:"error"`
	Suggestion  string    `json:"suggestion"`
	Retry       bool      `json:"retry"`
	ExitCode    int       `json:"exit_code"`
	WorkerID    string    `json:"worker_id"`
	ClaimedAt   time.Time `json:"claimed_at"`
	CompletedAt time.Time `json:"completed_at"`
	DurationMS  int64     `json:"duration_ms"`
}
