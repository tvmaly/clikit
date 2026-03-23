package filequeue

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureQueueDirs(t *testing.T) {
	root := t.TempDir()
	if err := EnsureQueueDirs(root); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{
		"pending", "claimed", "done", "dead",
		filepath.Join("payloads", "req"),
		filepath.Join("payloads", "resp"),
		"tmp",
	}
	for _, sub := range expected {
		p := filepath.Join(root, sub)
		fi, err := os.Stat(p)
		if err != nil {
			t.Errorf("expected dir %q to exist: %v", p, err)
			continue
		}
		if !fi.IsDir() {
			t.Errorf("expected %q to be a directory", p)
		}
	}
}

func TestEnsureQueueDirs_AlreadyExists(t *testing.T) {
	root := t.TempDir()
	if err := EnsureQueueDirs(root); err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if err := EnsureQueueDirs(root); err != nil {
		t.Fatalf("second call error (idempotent): %v", err)
	}
}

func TestQueuePaths(t *testing.T) {
	root := "/queue"
	p := QueuePaths(root)
	if p.Pending != filepath.Join(root, "pending") {
		t.Errorf("unexpected Pending path: %q", p.Pending)
	}
	if p.Claimed != filepath.Join(root, "claimed") {
		t.Errorf("unexpected Claimed path: %q", p.Claimed)
	}
	if p.Done != filepath.Join(root, "done") {
		t.Errorf("unexpected Done path: %q", p.Done)
	}
	if p.Dead != filepath.Join(root, "dead") {
		t.Errorf("unexpected Dead path: %q", p.Dead)
	}
	if p.PayloadsReq != filepath.Join(root, "payloads", "req") {
		t.Errorf("unexpected PayloadsReq path: %q", p.PayloadsReq)
	}
	if p.PayloadsResp != filepath.Join(root, "payloads", "resp") {
		t.Errorf("unexpected PayloadsResp path: %q", p.PayloadsResp)
	}
	if p.Tmp != filepath.Join(root, "tmp") {
		t.Errorf("unexpected Tmp path: %q", p.Tmp)
	}
}
