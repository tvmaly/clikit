package filequeue

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupQueue(t *testing.T) (root string, paths *Paths) {
	t.Helper()
	root = t.TempDir()
	if err := EnsureQueueDirs(root); err != nil {
		t.Fatalf("EnsureQueueDirs: %v", err)
	}
	return root, QueuePaths(root)
}

func writeRequestMeta(t *testing.T, dir, id string) {
	t.Helper()
	m := &RequestMeta{
		RequestID: id,
		Tool:      "test",
		Operation: "op",
		Method:    "GET",
		Path:      "/api/test",
		CreatedAt: time.Now().UTC(),
	}
	data, _ := json.Marshal(m)
	if err := AtomicWrite(dir, id+".meta.json", data); err != nil {
		t.Fatalf("writeRequestMeta: %v", err)
	}
}

func TestLifecycle_PendingToClaimed(t *testing.T) {
	root, p := setupQueue(t)
	_ = root
	id := "a1b2c3d4e5f67890"
	writeRequestMeta(t, p.Pending, id)

	if err := ClaimRequest(p, id); err != nil {
		t.Fatalf("ClaimRequest: %v", err)
	}
	if _, err := os.Stat(filepath.Join(p.Claimed, id+".meta.json")); err != nil {
		t.Error("expected file in claimed/")
	}
	if _, err := os.Stat(filepath.Join(p.Pending, id+".meta.json")); !os.IsNotExist(err) {
		t.Error("expected file gone from pending/")
	}
}

func TestLifecycle_ClaimedToDone(t *testing.T) {
	_, p := setupQueue(t)
	id := "a1b2c3d4e5f67890"
	writeRequestMeta(t, p.Claimed, id)

	payload := []byte(`{"result":"ok"}`)
	resp := &ResponseMeta{
		RequestID:   id,
		Status:      "success",
		StatusCode:  200,
		HasPayload:  true,
		PayloadPath: filepath.Join("payloads", "resp", id+".json"),
		ExitCode:    0,
		CompletedAt: time.Now().UTC(),
	}
	if err := CompleteRequest(p, id, payload, resp); err != nil {
		t.Fatalf("CompleteRequest: %v", err)
	}

	// Response meta in done/
	if _, err := os.Stat(filepath.Join(p.Done, id+".meta.json")); err != nil {
		t.Error("expected response meta in done/")
	}
	// Payload in payloads/resp/
	if _, err := os.Stat(filepath.Join(p.PayloadsResp, id+".json")); err != nil {
		t.Error("expected payload in payloads/resp/")
	}
	// Nothing in claimed/
	if _, err := os.Stat(filepath.Join(p.Claimed, id+".meta.json")); !os.IsNotExist(err) {
		t.Error("expected file gone from claimed/")
	}
}

func TestLifecycle_ClaimedToDead(t *testing.T) {
	_, p := setupQueue(t)
	id := "a1b2c3d4e5f67890"
	writeRequestMeta(t, p.Claimed, id)

	resp := &ResponseMeta{
		RequestID:  id,
		Status:     "error",
		StatusCode: 503,
		Error:      "handler not found",
		ExitCode:   1,
	}
	if err := DeadLetterRequest(p, id, resp); err != nil {
		t.Fatalf("DeadLetterRequest: %v", err)
	}
	if _, err := os.Stat(filepath.Join(p.Dead, id+".meta.json")); err != nil {
		t.Error("expected response meta in dead/")
	}
	if _, err := os.Stat(filepath.Join(p.Claimed, id+".meta.json")); !os.IsNotExist(err) {
		t.Error("expected file gone from claimed/")
	}
}

func TestLifecycle_DuplicateCheck(t *testing.T) {
	_, p := setupQueue(t)
	id := "a1b2c3d4e5f67890"
	// Write response meta to done/ (already processed).
	resp := &ResponseMeta{RequestID: id, Status: "success"}
	data, _ := json.Marshal(resp)
	AtomicWrite(p.Done, id+".meta.json", data)
	// Also write to pending/.
	writeRequestMeta(t, p.Pending, id)

	alreadyDone, err := IsAlreadyDone(p, id)
	if err != nil {
		t.Fatalf("IsAlreadyDone error: %v", err)
	}
	if !alreadyDone {
		t.Error("expected IsAlreadyDone=true")
	}
}

func TestLifecycle_FullCycle(t *testing.T) {
	_, p := setupQueue(t)
	id := "a1b2c3d4e5f67890"

	// Write to pending.
	writeRequestMeta(t, p.Pending, id)

	// Claim.
	if err := ClaimRequest(p, id); err != nil {
		t.Fatalf("ClaimRequest: %v", err)
	}

	// Complete.
	payload := []byte(`{"status":"done"}`)
	resp := &ResponseMeta{
		RequestID:  id,
		Status:     "success",
		StatusCode: 200,
		HasPayload: true,
	}
	if err := CompleteRequest(p, id, payload, resp); err != nil {
		t.Fatalf("CompleteRequest: %v", err)
	}

	// Verify final state.
	if _, err := os.Stat(filepath.Join(p.Done, id+".meta.json")); err != nil {
		t.Error("missing done meta")
	}
	if _, err := os.Stat(filepath.Join(p.Pending, id+".meta.json")); !os.IsNotExist(err) {
		t.Error("pending should be empty")
	}
	if _, err := os.Stat(filepath.Join(p.Claimed, id+".meta.json")); !os.IsNotExist(err) {
		t.Error("claimed should be empty")
	}
}
