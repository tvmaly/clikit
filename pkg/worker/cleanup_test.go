package worker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tvmaly/clikit/pkg/filequeue"
)

func writeResponseMeta(t *testing.T, dir, id string, completedAt time.Time) {
	t.Helper()
	m := &filequeue.ResponseMeta{
		RequestID:   id,
		Status:      "success",
		StatusCode:  200,
		HasPayload:  true,
		PayloadPath: filepath.Join("payloads", "resp", id+".json"),
		CompletedAt: completedAt,
	}
	data, _ := json.Marshal(m)
	filequeue.AtomicWrite(dir, id+".meta.json", data)
}

func TestCleanup_RemovesOldDone(t *testing.T) {
	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)
	p := filequeue.QueuePaths(root)

	id := "a1b2c3d4e5f67890"
	old := time.Now().Add(-48 * time.Hour)
	writeResponseMeta(t, p.Done, id, old)
	// Write corresponding payload.
	filequeue.AtomicWrite(p.PayloadsResp, id+".json", []byte(`{}`))

	cleaned, err := RunCleanup(root, 24*time.Hour, 72*time.Hour)
	if err != nil {
		t.Fatalf("RunCleanup error: %v", err)
	}
	if cleaned == 0 {
		t.Error("expected at least 1 file cleaned")
	}
	if _, err := os.Stat(filepath.Join(p.Done, id+".meta.json")); !os.IsNotExist(err) {
		t.Error("expected old done meta to be deleted")
	}
	if _, err := os.Stat(filepath.Join(p.PayloadsResp, id+".json")); !os.IsNotExist(err) {
		t.Error("expected old payload to be deleted")
	}
}

func TestCleanup_KeepsRecentDone(t *testing.T) {
	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)
	p := filequeue.QueuePaths(root)

	id := "a1b2c3d4e5f67890"
	writeResponseMeta(t, p.Done, id, time.Now().Add(-1*time.Hour))

	cleaned, err := RunCleanup(root, 24*time.Hour, 72*time.Hour)
	if err != nil {
		t.Fatalf("RunCleanup error: %v", err)
	}
	if cleaned != 0 {
		t.Errorf("expected 0 cleaned (recent file), got %d", cleaned)
	}
	if _, err := os.Stat(filepath.Join(p.Done, id+".meta.json")); err != nil {
		t.Error("expected recent done meta to remain")
	}
}

func TestCleanup_RemovesOldDead(t *testing.T) {
	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)
	p := filequeue.QueuePaths(root)

	id := "a1b2c3d4e5f67890"
	old := time.Now().Add(-80 * time.Hour)
	writeResponseMeta(t, p.Dead, id, old)

	cleaned, err := RunCleanup(root, 24*time.Hour, 72*time.Hour)
	if err != nil {
		t.Fatalf("RunCleanup error: %v", err)
	}
	if cleaned == 0 {
		t.Error("expected old dead meta to be cleaned")
	}
}

func TestCleanup_OrphanedTmp(t *testing.T) {
	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)
	p := filequeue.QueuePaths(root)

	// Write an old tmp file (backdated via modification time workaround:
	// just create it and set mtime in the past).
	tmpFile := filepath.Join(p.Tmp, "orphan.tmp")
	os.WriteFile(tmpFile, []byte("x"), 0o600)
	oldTime := time.Now().Add(-2 * time.Hour)
	os.Chtimes(tmpFile, oldTime, oldTime)

	cleaned, err := RunCleanup(root, 24*time.Hour, 72*time.Hour)
	if err != nil {
		t.Fatalf("RunCleanup error: %v", err)
	}
	if cleaned == 0 {
		t.Error("expected orphaned tmp to be cleaned")
	}
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("expected orphaned tmp file to be deleted")
	}
}

func TestCleanup_ReturnsCount(t *testing.T) {
	root := t.TempDir()
	filequeue.EnsureQueueDirs(root)
	p := filequeue.QueuePaths(root)

	old := time.Now().Add(-48 * time.Hour)
	ids := []string{"a1b2c3d4e5f67890", "b2c3d4e5f6789012"}
	for _, id := range ids {
		writeResponseMeta(t, p.Done, id, old)
		filequeue.AtomicWrite(p.PayloadsResp, id+".json", []byte(`{}`))
	}

	cleaned, err := RunCleanup(root, 24*time.Hour, 72*time.Hour)
	if err != nil {
		t.Fatalf("RunCleanup error: %v", err)
	}
	// 2 metas + 2 payloads = 4 files
	if cleaned < 4 {
		t.Errorf("expected at least 4 cleaned, got %d", cleaned)
	}
}
