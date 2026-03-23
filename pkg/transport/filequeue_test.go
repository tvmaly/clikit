package transport

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tvmaly/clikit/pkg/filequeue"
)

// simulateWorker reads the first pending request and writes a success response.
func simulateWorker(t *testing.T, queueRoot string, responseBody []byte) {
	t.Helper()
	p := filequeue.QueuePaths(queueRoot)

	// Poll for pending file.
	var id string
	for i := 0; i < 50; i++ {
		entries, _ := os.ReadDir(p.Pending)
		for _, e := range entries {
			if filepath.Ext(e.Name()) == ".json" {
				id = e.Name()[:len(e.Name())-len(".meta.json")]
				break
			}
		}
		if id != "" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if id == "" {
		t.Log("simulateWorker: no pending request found")
		return
	}

	// Claim it.
	if err := filequeue.AtomicMove(p.Pending, p.Claimed, id+".meta.json"); err != nil {
		t.Logf("simulateWorker: claim error: %v", err)
		return
	}

	// Write response payload.
	if err := filequeue.AtomicWrite(p.PayloadsResp, id+".json", responseBody); err != nil {
		t.Logf("simulateWorker: payload write error: %v", err)
		return
	}

	// Write response meta to done/.
	resp := &filequeue.ResponseMeta{
		RequestID:   id,
		Status:      "success",
		StatusCode:  200,
		HasPayload:  true,
		PayloadPath: filepath.Join("payloads", "resp", id+".json"),
		ExitCode:    0,
		CompletedAt: time.Now().UTC(),
	}
	data, _ := json.Marshal(resp)
	filequeue.AtomicWrite(p.Done, id+".meta.json", data)

	// Remove from claimed.
	os.Remove(filepath.Join(p.Claimed, id+".meta.json"))
}

func TestFileQueueTransport_Execute_Success(t *testing.T) {
	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)

	responseBody := []byte(`{"result":"ok"}`)
	go simulateWorker(t, root, responseBody)

	cfg := &Config{
		Mode:           "filequeue",
		QueueRoot:      root,
		PollInterval:   20 * time.Millisecond,
		RequestTimeout: 5 * time.Second,
	}
	tr, err := NewFileQueueTransport(cfg)
	if err != nil {
		t.Fatalf("NewFileQueueTransport: %v", err)
	}
	defer tr.Close()

	resp, err := tr.Execute(context.Background(), &Request{
		Tool: "test", Operation: "op", Method: "GET", Path: "/api/test",
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if string(resp.Body) != string(responseBody) {
		t.Errorf("expected body %q, got %q", responseBody, resp.Body)
	}
}

func TestFileQueueTransport_Execute_Timeout(t *testing.T) {
	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)

	cfg := &Config{
		Mode:           "filequeue",
		QueueRoot:      root,
		PollInterval:   20 * time.Millisecond,
		RequestTimeout: 200 * time.Millisecond,
	}
	tr, _ := NewFileQueueTransport(cfg)
	defer tr.Close()

	ctx := context.Background()
	resp, err := tr.Execute(ctx, &Request{Tool: "test", Operation: "op", Method: "GET", Path: "/"})
	if err == nil && (resp == nil || resp.Error == "") {
		t.Error("expected timeout error")
	}
}

func TestFileQueueTransport_Execute_DeadLetter(t *testing.T) {
	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)
	p := filequeue.QueuePaths(root)

	// Simulate a worker that moves request to dead-letter.
	go func() {
		for i := 0; i < 50; i++ {
			entries, _ := os.ReadDir(p.Pending)
			for _, e := range entries {
				if filepath.Ext(e.Name()) != ".json" {
					continue
				}
				id := e.Name()[:len(e.Name())-len(".meta.json")]
				filequeue.AtomicMove(p.Pending, p.Claimed, id+".meta.json")
				errResp := &filequeue.ResponseMeta{
					RequestID:  id,
					Status:     "error",
					StatusCode: 503,
					Error:      "dead-lettered",
					Retry:      false,
					ExitCode:   1,
				}
				data, _ := json.Marshal(errResp)
				filequeue.AtomicWrite(p.Dead, id+".meta.json", data)
				os.Remove(filepath.Join(p.Claimed, id+".meta.json"))
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()

	cfg := &Config{
		Mode:           "filequeue",
		QueueRoot:      root,
		PollInterval:   20 * time.Millisecond,
		RequestTimeout: 5 * time.Second,
	}
	tr, _ := NewFileQueueTransport(cfg)
	defer tr.Close()

	resp, err := tr.Execute(context.Background(), &Request{Tool: "test", Operation: "op", Method: "GET", Path: "/"})
	if err == nil && resp.Error == "" {
		t.Error("expected error for dead-letter response")
	}
}

func TestFileQueueTransport_Execute_WithPayload(t *testing.T) {
	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)
	p := filequeue.QueuePaths(root)

	go simulateWorker(t, root, []byte(`{"ok":true}`))

	cfg := &Config{
		Mode:           "filequeue",
		QueueRoot:      root,
		PollInterval:   20 * time.Millisecond,
		RequestTimeout: 5 * time.Second,
	}
	tr, _ := NewFileQueueTransport(cfg)
	defer tr.Close()

	body := []byte(`{"name":"test"}`)
	resp, err := tr.Execute(context.Background(), &Request{
		Tool: "test", Operation: "create", Method: "POST", Path: "/api/test", Body: body,
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Verify payload file was written in payloads/req/.
	entries, _ := os.ReadDir(p.PayloadsReq)
	if len(entries) == 0 {
		t.Error("expected request payload file in payloads/req/")
	}
}

func TestFileQueueTransport_Close(t *testing.T) {
	cfg := &Config{Mode: "filequeue", QueueRoot: t.TempDir()}
	tr, _ := NewFileQueueTransport(cfg)
	if err := tr.Close(); err != nil {
		t.Errorf("expected no error on Close, got %v", err)
	}
}
