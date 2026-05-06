package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/tvmaly/clikit/pkg/filequeue"
)

// Dispatcher scans the file queue for pending requests and processes them.
type Dispatcher struct {
	QueueRoot    string
	WorkerID     string
	Handlers     *HandlerConfig
	HTTPClient   *http.Client
	StaleClaim   time.Duration
	PollInterval time.Duration
	EventHook    EventHook
}

// Run is the main dispatch loop. It runs until ctx is cancelled.
func (d *Dispatcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(d.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			d.runOnce(ctx)
			RecoverStaleWithHook(d.QueueRoot, d.StaleClaim, d.EventHook) //nolint
		}
	}
}

// RunOnce performs one scan-claim-process cycle. It is primarily useful for
// command boundary tests and operational probes.
func (d *Dispatcher) RunOnce(ctx context.Context) {
	d.runOnce(ctx)
}

// runOnce performs one scan-claim-process cycle over pending/.
func (d *Dispatcher) runOnce(ctx context.Context) {
	p := filequeue.QueuePaths(d.QueueRoot)
	entries, err := os.ReadDir(p.Pending)
	if err != nil {
		return
	}

	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		// Extract and validate request ID.
		name := e.Name()
		id := name[:len(name)-len(".meta.json")]
		if !filequeue.SafeRequestID(id) {
			continue
		}

		d.processRequest(ctx, p, id)
	}
}

func (d *Dispatcher) processRequest(ctx context.Context, p *filequeue.Paths, id string) {
	// Idempotency: skip if already done.
	alreadyDone, _ := filequeue.IsAlreadyDone(p, id)
	if alreadyDone {
		os.Remove(filepath.Join(p.Pending, id+".meta.json"))
		return
	}

	// Read request meta.
	metaPath := filepath.Join(p.Pending, id+".meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return
	}
	var meta filequeue.RequestMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		// Malformed — dead-letter.
		d.writeDeadLetter(p, id, "parse error: "+err.Error())
		return
	}

	// Claim.
	if err := filequeue.ClaimRequest(p, id); err != nil {
		return // another worker claimed it
	}
	WriteHeartbeat(p.Claimed, id, d.WorkerID) //nolint
	d.emit(Event{Name: "request_claimed", RequestID: id, Tool: meta.Tool, Operation: meta.Operation, State: "claimed"})

	// Look up handler.
	handler, ok := d.Handlers.Lookup(meta.Tool, meta.Operation)
	if !ok {
		d.writeErrorResponse(p, id, fmt.Sprintf("no handler for tool=%q operation=%q", meta.Tool, meta.Operation), false, 404)
		return
	}
	d.emit(Event{Name: "handler_selected", RequestID: id, Tool: meta.Tool, Operation: meta.Operation, State: "claimed"})

	// Build URL with path params.
	finalURL, remainingParams, err := BuildURL(handler, meta.Params)
	if err != nil {
		d.writeErrorResponse(p, id, "building URL: "+err.Error(), false, 400)
		return
	}

	// Execute HTTP request.
	d.executeRequest(ctx, p, id, &meta, handler, finalURL, remainingParams)
}

func (d *Dispatcher) executeRequest(
	ctx context.Context,
	p *filequeue.Paths,
	id string,
	meta *filequeue.RequestMeta,
	handler *HandlerEntry,
	baseURL string,
	params map[string]string,
) {
	start := time.Now()

	// Build URL with query params.
	u, err := url.Parse(baseURL)
	if err != nil {
		d.writeErrorResponse(p, id, "invalid URL: "+err.Error(), false, 400)
		return
	}
	if len(params) > 0 {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	// Read request payload if present.
	var bodyReader io.Reader
	if meta.HasPayload {
		payload, err := os.ReadFile(filepath.Join(d.QueueRoot, meta.PayloadPath))
		if err != nil {
			d.writeErrorResponse(p, id, "reading request payload: "+err.Error(), false, 500)
			return
		}
		bodyReader = bytes.NewReader(payload)
	}

	timeout := time.Duration(handler.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, meta.Method, u.String(), bodyReader)
	if err != nil {
		d.writeErrorResponse(p, id, "creating request: "+err.Error(), false, 500)
		return
	}
	if handler.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+handler.Token)
	}
	if meta.HasPayload {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	d.emit(Event{Name: "http_request_started", RequestID: id, Tool: meta.Tool, Operation: meta.Operation, State: "claimed"})
	httpResp, err := client.Do(httpReq)
	if err != nil {
		d.writeErrorResponse(p, id, "http request failed: "+err.Error(), true, 0)
		d.emit(Event{Name: "request_dead_lettered", RequestID: id, Tool: meta.Tool, Operation: meta.Operation, State: "done", Retry: true, ErrorClass: "network"})
		return
	}
	defer httpResp.Body.Close()
	d.emit(Event{Name: "http_response_received", RequestID: id, Tool: meta.Tool, Operation: meta.Operation, State: "claimed", DurationMS: durationMS(start)})

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		d.writeErrorResponse(p, id, "reading response: "+err.Error(), true, httpResp.StatusCode)
		return
	}

	dur := time.Since(start)
	retry := httpResp.StatusCode >= 500

	if httpResp.StatusCode >= 400 {
		suggestion := ""
		if httpResp.StatusCode >= 500 {
			suggestion = "The API may be temporarily unavailable. Retry later."
		}
		resp := &filequeue.ResponseMeta{
			RequestID:   id,
			Status:      "error",
			StatusCode:  httpResp.StatusCode,
			Error:       fmt.Sprintf("API returned %d", httpResp.StatusCode),
			Suggestion:  suggestion,
			Retry:       retry,
			ExitCode:    1,
			WorkerID:    d.WorkerID,
			CompletedAt: time.Now().UTC(),
			DurationMS:  dur.Milliseconds(),
		}
		filequeue.CompleteRequest(p, id, respBody, resp) //nolint
		d.emit(Event{Name: "request_completed", RequestID: id, Tool: meta.Tool, Operation: meta.Operation, State: "done", DurationMS: dur.Milliseconds(), Retry: retry, ErrorClass: "http_status"})
		return
	}

	resp := &filequeue.ResponseMeta{
		RequestID:   id,
		Status:      "success",
		StatusCode:  httpResp.StatusCode,
		ExitCode:    0,
		WorkerID:    d.WorkerID,
		CompletedAt: time.Now().UTC(),
		DurationMS:  dur.Milliseconds(),
	}
	filequeue.CompleteRequest(p, id, respBody, resp) //nolint
	d.emit(Event{Name: "request_completed", RequestID: id, Tool: meta.Tool, Operation: meta.Operation, State: "done", DurationMS: dur.Milliseconds(), ErrorClass: "none"})
}

func (d *Dispatcher) writeErrorResponse(p *filequeue.Paths, id, msg string, retry bool, statusCode int) {
	resp := &filequeue.ResponseMeta{
		RequestID:   id,
		Status:      "error",
		StatusCode:  statusCode,
		Error:       msg,
		Retry:       retry,
		ExitCode:    1,
		WorkerID:    d.WorkerID,
		CompletedAt: time.Now().UTC(),
	}
	filequeue.CompleteRequest(p, id, nil, resp) //nolint
	d.emit(Event{Name: "request_completed", RequestID: id, State: "done", Retry: retry, ErrorClass: "worker_error"})
}

func (d *Dispatcher) writeDeadLetter(p *filequeue.Paths, id, msg string) {
	// Move from pending to claimed first (claim it), then dead-letter.
	filequeue.AtomicMove(p.Pending, p.Claimed, id+".meta.json") //nolint
	resp := &filequeue.ResponseMeta{
		RequestID:   id,
		Status:      "error",
		Error:       msg,
		ExitCode:    1,
		WorkerID:    d.WorkerID,
		CompletedAt: time.Now().UTC(),
	}
	filequeue.DeadLetterRequest(p, id, resp) //nolint
	d.emit(Event{Name: "request_dead_lettered", RequestID: id, State: "dead", ErrorClass: "malformed_request"})
}
