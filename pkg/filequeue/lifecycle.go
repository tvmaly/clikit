package filequeue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ClaimRequest atomically moves the request from pending/ to claimed/.
// Returns ErrAlreadyClaimed if another worker already claimed it.
func ClaimRequest(p *Paths, id string) error {
	return AtomicMove(p.Pending, p.Claimed, id+".meta.json")
}

// CompleteRequest writes the response payload, writes the response metadata to
// done/, and removes the claimed/ entry. It is the transition: claimed → done.
func CompleteRequest(p *Paths, id string, payload []byte, resp *ResponseMeta) error {
	// Write payload if provided.
	if len(payload) > 0 {
		if err := AtomicWrite(p.PayloadsResp, id+".json", payload); err != nil {
			return fmt.Errorf("writing response payload: %w", err)
		}
		resp.PayloadPath = filepath.Join("payloads", "resp", id+".json")
		resp.HasPayload = true
	}

	// Write response metadata to done/.
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshaling response meta: %w", err)
	}
	if err := AtomicWrite(p.Done, id+".meta.json", data); err != nil {
		return fmt.Errorf("writing response meta: %w", err)
	}

	// Remove from claimed/ (best-effort; response meta is the definitive signal).
	os.Remove(filepath.Join(p.Claimed, id+".meta.json"))
	return nil
}

// DeadLetterRequest writes the error response metadata to dead/ and removes
// the claimed/ entry. It is the transition: claimed → dead.
func DeadLetterRequest(p *Paths, id string, resp *ResponseMeta) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshaling dead-letter meta: %w", err)
	}
	if err := AtomicWrite(p.Dead, id+".meta.json", data); err != nil {
		return fmt.Errorf("writing dead-letter meta: %w", err)
	}
	os.Remove(filepath.Join(p.Claimed, id+".meta.json"))
	return nil
}

// IsAlreadyDone reports whether a response meta file already exists in done/
// for the given request ID, indicating the request was already processed.
func IsAlreadyDone(p *Paths, id string) (bool, error) {
	_, err := os.Stat(filepath.Join(p.Done, id+".meta.json"))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("checking done/ for %q: %w", id, err)
}
