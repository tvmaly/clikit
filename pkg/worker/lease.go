package worker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tvmaly/clikit/pkg/filequeue"
)

// Heartbeat is the heartbeat file written to claimed/<id>.heartbeat.
type Heartbeat struct {
	WorkerID      string    `json:"worker_id"`
	ClaimedAt     time.Time `json:"claimed_at"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

// WriteHeartbeat writes a new heartbeat file for the given request.
func WriteHeartbeat(claimedDir, requestID, workerID string) error {
	hb := Heartbeat{
		WorkerID:      workerID,
		ClaimedAt:     time.Now().UTC(),
		LastHeartbeat: time.Now().UTC(),
	}
	data, err := json.Marshal(hb)
	if err != nil {
		return fmt.Errorf("marshaling heartbeat: %w", err)
	}
	return filequeue.AtomicWrite(claimedDir, requestID+".heartbeat", data)
}

// UpdateHeartbeat reads an existing heartbeat file and updates LastHeartbeat.
func UpdateHeartbeat(heartbeatPath string) error {
	data, err := os.ReadFile(heartbeatPath)
	if err != nil {
		return fmt.Errorf("reading heartbeat: %w", err)
	}
	var hb Heartbeat
	if err := json.Unmarshal(data, &hb); err != nil {
		return fmt.Errorf("parsing heartbeat: %w", err)
	}
	hb.LastHeartbeat = time.Now().UTC()
	updated, err := json.Marshal(hb)
	if err != nil {
		return fmt.Errorf("marshaling updated heartbeat: %w", err)
	}
	dir := filepath.Dir(heartbeatPath)
	name := filepath.Base(heartbeatPath)
	return filequeue.AtomicWrite(dir, name, updated)
}

// IsStale reports whether the heartbeat at heartbeatPath is older than threshold.
func IsStale(heartbeatPath string, threshold time.Duration) (bool, error) {
	data, err := os.ReadFile(heartbeatPath)
	if err != nil {
		return false, fmt.Errorf("reading heartbeat: %w", err)
	}
	var hb Heartbeat
	if err := json.Unmarshal(data, &hb); err != nil {
		return false, fmt.Errorf("parsing heartbeat: %w", err)
	}
	return time.Since(hb.LastHeartbeat) > threshold, nil
}

// RecoverStale scans claimed/ for stale heartbeats and moves their requests
// back to pending/. Returns the list of recovered request IDs.
func RecoverStale(queueRoot string, threshold time.Duration) ([]string, error) {
	return RecoverStaleWithHook(queueRoot, threshold, nil)
}

// RecoverStaleWithHook is RecoverStale with optional structured event emission.
func RecoverStaleWithHook(queueRoot string, threshold time.Duration, hook EventHook) ([]string, error) {
	p := filequeue.QueuePaths(queueRoot)
	entries, err := os.ReadDir(p.Claimed)
	if err != nil {
		return nil, fmt.Errorf("reading claimed dir: %w", err)
	}

	var recovered []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".heartbeat" {
			continue
		}
		hbPath := filepath.Join(p.Claimed, e.Name())
		stale, err := IsStale(hbPath, threshold)
		if err != nil || !stale {
			continue
		}
		// Extract request ID: strip ".heartbeat" suffix.
		id := e.Name()[:len(e.Name())-len(".heartbeat")]
		if !filequeue.SafeRequestID(id) {
			continue
		}

		// Remove heartbeat file.
		os.Remove(hbPath)

		// Move request meta back to pending/.
		if err := filequeue.AtomicMove(p.Claimed, p.Pending, id+".meta.json"); err != nil {
			// Already gone or claimed — skip.
			continue
		}
		recovered = append(recovered, id)
		if hook != nil {
			hook(Event{Name: "stale_claim_recovered", RequestID: id, State: "pending", ErrorClass: "stale_claim"})
		}
	}
	return recovered, nil
}
