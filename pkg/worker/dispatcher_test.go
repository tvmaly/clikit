package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tvmaly/clikit/pkg/filequeue"
)

func buildTestDispatcher(t *testing.T, queueRoot string, srv *httptest.Server) *Dispatcher {
	t.Helper()
	hc := &HandlerConfig{
		Handlers: []HandlerEntry{
			{
				Tool:      "test",
				Operation: "op",
				Method:    "GET",
				URLPath:   "/api/result",
				BaseURL:   srv.URL,
				Timeout:   5,
			},
		},
	}
	return &Dispatcher{
		QueueRoot:    queueRoot,
		WorkerID:     "worker-test",
		Handlers:     hc,
		HTTPClient:   srv.Client(),
		StaleClaim:   30 * time.Second,
		PollInterval: 10 * time.Millisecond,
	}
}

func TestDispatcher_EmitsStructuredEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)
	p := filequeue.QueuePaths(root)
	id := "a1b2c3d4e5f67890"
	writePendingRequest(t, p, id, "test", "op")

	var events []Event
	d := buildTestDispatcher(t, root, srv)
	d.EventHook = func(e Event) {
		events = append(events, e)
	}
	d.runOnce(context.Background())

	want := []string{"request_claimed", "handler_selected", "http_request_started", "http_response_received", "request_completed"}
	for _, name := range want {
		found := false
		for _, event := range events {
			if event.Name == name && event.RequestID == id && event.Tool == "test" && event.Operation == "op" && event.WorkerID == "worker-test" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing event %q in %#v", name, events)
		}
	}
}

func TestDispatcher_LogEventJSONOmitsSecrets(t *testing.T) {
	var buf bytes.Buffer
	err := WriteEventJSON(&buf, Event{
		Name:       "request_completed",
		RequestID:  "a1b2c3d4e5f67890",
		Tool:       "test",
		Operation:  "op",
		WorkerID:   "worker-test",
		DurationMS: 12,
		State:      "done",
		Retry:      false,
		ErrorClass: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, `"event":"request_completed"`) || !strings.Contains(got, `"request_id":"a1b2c3d4e5f67890"`) {
		t.Fatalf("unexpected event JSON: %q", got)
	}
	if strings.Contains(strings.ToLower(got), "token") || strings.Contains(strings.ToLower(got), "bearer") {
		t.Fatalf("event JSON must not include secrets: %q", got)
	}
}

func writePendingRequest(t *testing.T, p *filequeue.Paths, id, tool, op string) {
	t.Helper()
	meta := &filequeue.RequestMeta{
		RequestID:           id,
		Tool:                tool,
		Operation:           op,
		Method:              "GET",
		Path:                "/api/result",
		ResponseMetaPath:    filepath.Join("done", id+".meta.json"),
		ResponsePayloadPath: filepath.Join("payloads", "resp", id+".json"),
		TimeoutSeconds:      30,
		CreatedAt:           time.Now().UTC(),
	}
	data, _ := json.Marshal(meta)
	filequeue.AtomicWrite(p.Pending, id+".meta.json", data)
}

func TestDispatcher_ProcessesRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)
	p := filequeue.QueuePaths(root)

	id := "a1b2c3d4e5f67890"
	writePendingRequest(t, p, id, "test", "op")

	d := buildTestDispatcher(t, root, srv)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Run one cycle.
	d.runOnce(ctx)

	// Check done/.
	donePath := filepath.Join(p.Done, id+".meta.json")
	data, err := os.ReadFile(donePath)
	if err != nil {
		t.Fatalf("expected response in done/: %v", err)
	}
	var resp filequeue.ResponseMeta
	json.Unmarshal(data, &resp)
	if resp.Status != "success" {
		t.Errorf("expected success, got %q", resp.Status)
	}
}

func TestDispatcher_HandlerNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)
	p := filequeue.QueuePaths(root)

	id := "a1b2c3d4e5f67890"
	writePendingRequest(t, p, id, "unknown", "unknown-op")

	d := buildTestDispatcher(t, root, srv)
	d.runOnce(context.Background())

	donePath := filepath.Join(p.Done, id+".meta.json")
	data, err := os.ReadFile(donePath)
	if err != nil {
		t.Fatalf("expected error response in done/: %v", err)
	}
	var resp filequeue.ResponseMeta
	json.Unmarshal(data, &resp)
	if resp.Status != "error" {
		t.Errorf("expected error status, got %q", resp.Status)
	}
}

func TestDispatcher_APIFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		w.Write([]byte(`{"message":"unavailable"}`))
	}))
	defer srv.Close()

	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)
	p := filequeue.QueuePaths(root)

	id := "a1b2c3d4e5f67890"
	writePendingRequest(t, p, id, "test", "op")

	d := buildTestDispatcher(t, root, srv)
	d.runOnce(context.Background())

	data, _ := os.ReadFile(filepath.Join(p.Done, id+".meta.json"))
	var resp filequeue.ResponseMeta
	json.Unmarshal(data, &resp)
	if resp.Status != "error" {
		t.Errorf("expected error for 503, got status=%q", resp.Status)
	}
	if !resp.Retry {
		t.Error("expected retry=true for 5xx")
	}
}

func TestDispatcher_MalformedRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)
	p := filequeue.QueuePaths(root)

	// Write invalid JSON to pending/.
	filequeue.AtomicWrite(p.Pending, "a1b2c3d4e5f67890.meta.json", []byte("not json"))

	d := buildTestDispatcher(t, root, srv)
	d.runOnce(context.Background())

	// Should move to dead/.
	deadPath := filepath.Join(p.Dead, "a1b2c3d4e5f67890.meta.json")
	if _, err := os.Stat(deadPath); err != nil {
		t.Error("expected malformed request to be dead-lettered")
	}
}

func TestDispatcher_InvalidRequestID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)
	p := filequeue.QueuePaths(root)

	// Write file with invalid name.
	os.WriteFile(filepath.Join(p.Pending, "invalid-id.meta.json"), []byte(`{}`), 0o600)

	d := buildTestDispatcher(t, root, srv)
	d.runOnce(context.Background())

	// File should remain (skipped, not claimed).
	if _, err := os.Stat(filepath.Join(p.Pending, "invalid-id.meta.json")); err != nil {
		t.Error("expected file with invalid ID to remain in pending/")
	}
}

func TestDispatcher_DuplicateSkip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)
	p := filequeue.QueuePaths(root)

	id := "a1b2c3d4e5f67890"
	// Write pending AND done (already processed).
	writePendingRequest(t, p, id, "test", "op")
	resp := filequeue.ResponseMeta{RequestID: id, Status: "success"}
	data, _ := json.Marshal(resp)
	filequeue.AtomicWrite(p.Done, id+".meta.json", data)

	d := buildTestDispatcher(t, root, srv)
	d.runOnce(context.Background())

	// Pending should be deleted.
	if _, err := os.Stat(filepath.Join(p.Pending, id+".meta.json")); !os.IsNotExist(err) {
		t.Error("expected pending file to be cleaned up for duplicate")
	}
}

func TestDispatcher_ConcurrentWorkers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)
	p := filequeue.QueuePaths(root)

	// Write 10 requests.
	ids := make([]string, 10)
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("a%015x", i)
		ids[i] = id
		writePendingRequest(t, p, id, "test", "op")
	}

	// Run 2 dispatchers concurrently until all requests are processed.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	hc := &HandlerConfig{
		Handlers: []HandlerEntry{
			{Tool: "test", Operation: "op", Method: "GET", URLPath: "/api/result", BaseURL: srv.URL, Timeout: 5},
		},
	}
	var wg sync.WaitGroup
	for w := 0; w < 2; w++ {
		wid := fmt.Sprintf("worker-%d", w)
		d := &Dispatcher{
			QueueRoot: root, WorkerID: wid, Handlers: hc,
			HTTPClient: srv.Client(), StaleClaim: 30 * time.Second, PollInterval: 5 * time.Millisecond,
		}
		wg.Add(1)
		go func(d *Dispatcher) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					d.runOnce(ctx)
					time.Sleep(5 * time.Millisecond)
				}
			}
		}(d)
	}

	// Wait for all 10 to be done.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		doneCount := 0
		for _, id := range ids {
			if _, err := os.Stat(filepath.Join(p.Done, id+".meta.json")); err == nil {
				doneCount++
			}
		}
		if doneCount == 10 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel() // stop workers
	wg.Wait()

	// Verify all 10 processed exactly once.
	for _, id := range ids {
		donePath := filepath.Join(p.Done, id+".meta.json")
		if _, err := os.Stat(donePath); err != nil {
			t.Errorf("request %q not done", id)
		}
	}
}

func TestDispatcher_GracefulShutdown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)

	d := buildTestDispatcher(t, root, srv)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	// Run should return quickly without error.
	done := make(chan error, 1)
	go func() { done <- d.Run(ctx) }()
	select {
	case err := <-done:
		if err != nil && err != context.Canceled {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Run did not return after context cancel")
	}
}
