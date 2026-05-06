package worker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tvmaly/clikit/pkg/filequeue"
)

func setupLeaseQueue(t *testing.T) (root string, paths *filequeue.Paths) {
	t.Helper()
	root = t.TempDir()
	filequeue.EnsureQueueDirs(root)
	return root, filequeue.QueuePaths(root)
}

func TestWriteHeartbeat(t *testing.T) {
	root, p := setupLeaseQueue(t)
	_ = root
	id := "a1b2c3d4e5f67890"
	if err := WriteHeartbeat(p.Claimed, id, "worker-01"); err != nil {
		t.Fatalf("WriteHeartbeat error: %v", err)
	}
	hbPath := filepath.Join(p.Claimed, id+".heartbeat")
	data, err := os.ReadFile(hbPath)
	if err != nil {
		t.Fatalf("reading heartbeat: %v", err)
	}
	var hb Heartbeat
	if err := json.Unmarshal(data, &hb); err != nil {
		t.Fatalf("parsing heartbeat: %v", err)
	}
	if hb.WorkerID != "worker-01" {
		t.Errorf("expected worker_id='worker-01', got %q", hb.WorkerID)
	}
}

func TestUpdateHeartbeat(t *testing.T) {
	_, p := setupLeaseQueue(t)
	id := "a1b2c3d4e5f67890"
	WriteHeartbeat(p.Claimed, id, "worker-01")

	time.Sleep(10 * time.Millisecond)
	hbPath := filepath.Join(p.Claimed, id+".heartbeat")
	if err := UpdateHeartbeat(hbPath); err != nil {
		t.Fatalf("UpdateHeartbeat error: %v", err)
	}

	data, _ := os.ReadFile(hbPath)
	var hb Heartbeat
	json.Unmarshal(data, &hb)
	if !hb.LastHeartbeat.After(hb.ClaimedAt) {
		t.Error("expected last_heartbeat to be after claimed_at after update")
	}
}

func TestIsStale_Fresh(t *testing.T) {
	_, p := setupLeaseQueue(t)
	id := "a1b2c3d4e5f67890"
	WriteHeartbeat(p.Claimed, id, "worker-01")
	hbPath := filepath.Join(p.Claimed, id+".heartbeat")
	stale, err := IsStale(hbPath, 5*time.Minute)
	if err != nil {
		t.Fatalf("IsStale error: %v", err)
	}
	if stale {
		t.Error("expected fresh heartbeat to not be stale")
	}
}

func TestIsStale_Expired(t *testing.T) {
	_, p := setupLeaseQueue(t)
	id := "a1b2c3d4e5f67890"
	// Write heartbeat with old timestamp.
	hb := Heartbeat{
		WorkerID:      "worker-01",
		ClaimedAt:     time.Now().Add(-20 * time.Minute),
		LastHeartbeat: time.Now().Add(-10 * time.Minute),
	}
	data, _ := json.Marshal(hb)
	filequeue.AtomicWrite(p.Claimed, id+".heartbeat", data)

	hbPath := filepath.Join(p.Claimed, id+".heartbeat")
	stale, err := IsStale(hbPath, 5*time.Minute)
	if err != nil {
		t.Fatalf("IsStale error: %v", err)
	}
	if !stale {
		t.Error("expected stale heartbeat to be detected")
	}
}

func TestRecoverStale(t *testing.T) {
	root, p := setupLeaseQueue(t)
	id := "a1b2c3d4e5f67890"

	// Write request meta to claimed/.
	meta := map[string]any{"request_id": id}
	data, _ := json.Marshal(meta)
	filequeue.AtomicWrite(p.Claimed, id+".meta.json", data)

	// Write stale heartbeat.
	hb := Heartbeat{
		WorkerID:      "worker-01",
		ClaimedAt:     time.Now().Add(-20 * time.Minute),
		LastHeartbeat: time.Now().Add(-10 * time.Minute),
	}
	hbData, _ := json.Marshal(hb)
	filequeue.AtomicWrite(p.Claimed, id+".heartbeat", hbData)

	recovered, err := RecoverStale(root, 5*time.Minute)
	if err != nil {
		t.Fatalf("RecoverStale error: %v", err)
	}
	if len(recovered) != 1 || recovered[0] != id {
		t.Errorf("expected [%q] recovered, got %v", id, recovered)
	}
	// Request should be back in pending/.
	if _, err := os.Stat(filepath.Join(p.Pending, id+".meta.json")); err != nil {
		t.Error("expected request back in pending/")
	}
	// Heartbeat should be deleted.
	if _, err := os.Stat(filepath.Join(p.Claimed, id+".heartbeat")); !os.IsNotExist(err) {
		t.Error("expected heartbeat to be deleted")
	}
}

func TestRecoverStaleWithHookEmitsEvents(t *testing.T) {
	root, paths := setupLeaseQueue(t)
	id := "a1b2c3d4e5f67890"
	// Write claimed request meta.
	meta := &filequeue.RequestMeta{RequestID: id}
	metaData, _ := json.Marshal(meta)
	filequeue.AtomicWrite(paths.Claimed, id+".meta.json", metaData)
	hb := Heartbeat{
		WorkerID:      "worker-1",
		ClaimedAt:     time.Now().Add(-10 * time.Minute),
		LastHeartbeat: time.Now().Add(-10 * time.Minute),
	}
	data, _ := json.Marshal(hb)
	filequeue.AtomicWrite(paths.Claimed, id+".heartbeat", data)

	var events []Event
	recovered, err := RecoverStaleWithHook(root, 5*time.Minute, func(e Event) {
		events = append(events, e)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered) != 1 || recovered[0] != id {
		t.Fatalf("expected recovered %q, got %#v", id, recovered)
	}
	for _, e := range events {
		if e.Name == "stale_claim_recovered" && e.RequestID == id && e.State == "pending" {
			return
		}
	}
	t.Fatalf("expected stale_claim_recovered event, got %#v", events)
}

func TestRecoverStale_FreshSkipped(t *testing.T) {
	root, p := setupLeaseQueue(t)
	id := "a1b2c3d4e5f67890"

	meta := map[string]any{"request_id": id}
	data, _ := json.Marshal(meta)
	filequeue.AtomicWrite(p.Claimed, id+".meta.json", data)
	WriteHeartbeat(p.Claimed, id, "worker-01")

	recovered, err := RecoverStale(root, 5*time.Minute)
	if err != nil {
		t.Fatalf("RecoverStale error: %v", err)
	}
	if len(recovered) != 0 {
		t.Errorf("expected 0 recovered, got %v", recovered)
	}
}
