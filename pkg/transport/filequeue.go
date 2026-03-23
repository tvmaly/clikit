package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tvmaly/clikit/pkg/filequeue"
)

// FileQueueTransport implements Transport using the file-queue mechanism.
type FileQueueTransport struct {
	queueRoot      string
	pollInterval   time.Duration
	requestTimeout time.Duration
	paths          *filequeue.Paths
}

// NewFileQueueTransport creates a FileQueueTransport from cfg.
// It ensures the queue directories exist.
func NewFileQueueTransport(cfg *Config) (*FileQueueTransport, error) {
	poll := cfg.PollInterval
	if poll == 0 {
		poll = DefaultConfig().PollInterval
	}
	timeout := cfg.RequestTimeout
	if timeout == 0 {
		timeout = DefaultConfig().RequestTimeout
	}
	p := filequeue.QueuePaths(cfg.QueueRoot)
	if err := filequeue.EnsureQueueDirs(cfg.QueueRoot); err != nil {
		return nil, fmt.Errorf("ensuring queue dirs: %w", err)
	}
	return &FileQueueTransport{
		queueRoot:      cfg.QueueRoot,
		pollInterval:   poll,
		requestTimeout: timeout,
		paths:          p,
	}, nil
}

// Execute writes a request to the queue, polls for the response, and returns it.
func (t *FileQueueTransport) Execute(ctx context.Context, req *Request) (*Response, error) {
	id, err := filequeue.GenerateRequestID()
	if err != nil {
		return nil, fmt.Errorf("generating request ID: %w", err)
	}

	meta := &filequeue.RequestMeta{
		RequestID:           id,
		Tool:                req.Tool,
		Operation:           req.Operation,
		Method:              req.Method,
		Path:                req.Path,
		Params:              req.Params,
		HasPayload:          len(req.Body) > 0,
		ResponseMetaPath:    filepath.Join("done", id+".meta.json"),
		ResponsePayloadPath: filepath.Join("payloads", "resp", id+".json"),
		TimeoutSeconds:      int(t.requestTimeout.Seconds()),
		CreatedAt:           time.Now().UTC(),
		ExpectedResponse:    "json",
	}

	// Write request payload if present.
	if len(req.Body) > 0 {
		if err := filequeue.AtomicWrite(t.paths.PayloadsReq, id+".json", req.Body); err != nil {
			return nil, fmt.Errorf("writing request payload: %w", err)
		}
		meta.PayloadPath = filepath.Join("payloads", "req", id+".json")
	}

	// Write request metadata to pending/.
	metaData, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshaling request meta: %w", err)
	}
	if err := filequeue.AtomicWrite(t.paths.Pending, id+".meta.json", metaData); err != nil {
		return nil, fmt.Errorf("writing request meta: %w", err)
	}

	// Poll for response with timeout.
	timeoutCtx, cancel := context.WithTimeout(ctx, t.requestTimeout)
	defer cancel()

	respMeta, err := t.waitForResponse(timeoutCtx, id)
	if err != nil {
		return &Response{Error: err.Error()}, err
	}

	// Read response payload.
	var body []byte
	if respMeta.HasPayload {
		payloadPath := filepath.Join(t.queueRoot, respMeta.PayloadPath)
		body, err = os.ReadFile(payloadPath)
		if err != nil {
			return nil, fmt.Errorf("reading response payload: %w", err)
		}
	}

	if respMeta.Status == "error" {
		return &Response{
			StatusCode: respMeta.StatusCode,
			Body:       body,
			Error:      respMeta.Error,
		}, nil
	}

	return &Response{StatusCode: respMeta.StatusCode, Body: body}, nil
}

// waitForResponse polls done/ and dead/ until a response appears or ctx is cancelled.
func (t *FileQueueTransport) waitForResponse(ctx context.Context, id string) (*filequeue.ResponseMeta, error) {
	donePath := filepath.Join(t.queueRoot, "done", id+".meta.json")
	deadPath := filepath.Join(t.queueRoot, "dead", id+".meta.json")

	ticker := time.NewTicker(t.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("request %q timed out: %w", id, ctx.Err())
		case <-ticker.C:
			// Check done/.
			if data, err := os.ReadFile(donePath); err == nil {
				var meta filequeue.ResponseMeta
				if err := json.Unmarshal(data, &meta); err != nil {
					return nil, fmt.Errorf("parsing done response: %w", err)
				}
				return &meta, nil
			}
			// Check dead/.
			if data, err := os.ReadFile(deadPath); err == nil {
				var meta filequeue.ResponseMeta
				if err := json.Unmarshal(data, &meta); err != nil {
					return nil, fmt.Errorf("parsing dead-letter response: %w", err)
				}
				return nil, errors.New(meta.Error)
			}
		}
	}
}

// Close is a no-op for FileQueueTransport.
func (t *FileQueueTransport) Close() error { return nil }
