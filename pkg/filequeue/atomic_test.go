package filequeue

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWrite_Success(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`{"key":"value"}`)
	if err := AtomicWrite(dir, "test.json", data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "test.json"))
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("expected %q, got %q", data, got)
	}
	// No temp files should remain.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("temp file left behind: %q", e.Name())
		}
	}
}

func TestAtomicWrite_ContentMatch(t *testing.T) {
	dir := t.TempDir()
	data := make([]byte, 10240)
	for i := range data {
		data[i] = byte('a' + i%26)
	}
	if err := AtomicWrite(dir, "big.json", data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "big.json"))
	if string(got) != string(data) {
		t.Error("content mismatch after AtomicWrite")
	}
}

func TestAtomicWrite_CreatesParentDir(t *testing.T) {
	// AtomicWrite must NOT create parent dirs; caller is responsible.
	nonExistent := filepath.Join(t.TempDir(), "missing_subdir")
	err := AtomicWrite(nonExistent, "file.json", []byte("x"))
	if err == nil {
		t.Error("expected error when parent dir does not exist")
	}
}

func TestAtomicMove_Success(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	// Write a file to srcDir.
	if err := AtomicWrite(srcDir, "req.meta.json", []byte("{}")); err != nil {
		t.Fatal(err)
	}
	if err := AtomicMove(srcDir, dstDir, "req.meta.json"); err != nil {
		t.Fatalf("AtomicMove error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dstDir, "req.meta.json")); err != nil {
		t.Error("file not in dest dir")
	}
	if _, err := os.Stat(filepath.Join(srcDir, "req.meta.json")); !os.IsNotExist(err) {
		t.Error("file should be gone from src dir")
	}
}

func TestAtomicMove_AlreadyClaimed(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	AtomicWrite(srcDir, "req.meta.json", []byte("{}"))
	// First move succeeds.
	if err := AtomicMove(srcDir, dstDir, "req.meta.json"); err != nil {
		t.Fatalf("first move failed: %v", err)
	}
	// Second move on same file (no longer in src) returns ErrAlreadyClaimed.
	err := AtomicMove(srcDir, dstDir, "req.meta.json")
	if err != ErrAlreadyClaimed {
		t.Errorf("expected ErrAlreadyClaimed, got %v", err)
	}
}

func TestSafeRequestID_Valid(t *testing.T) {
	if !SafeRequestID("a1b2c3d4e5f67890") {
		t.Error("expected true for valid ID")
	}
}

func TestSafeRequestID_TooShort(t *testing.T) {
	if SafeRequestID("a1b2c3") {
		t.Error("expected false for short ID")
	}
}

func TestSafeRequestID_TooLong(t *testing.T) {
	if SafeRequestID("a1b2c3d4e5f678901234") {
		t.Error("expected false for long ID")
	}
}

func TestSafeRequestID_UpperCase(t *testing.T) {
	if SafeRequestID("A1B2C3D4E5F67890") {
		t.Error("expected false for uppercase ID")
	}
}

func TestSafeRequestID_PathTraversal(t *testing.T) {
	if SafeRequestID("../../../etc/passwd") {
		t.Error("expected false for path traversal")
	}
}

func TestSafeRequestID_WithSlashes(t *testing.T) {
	if SafeRequestID("a1b2/c3d4e5f6789") {
		t.Error("expected false for ID with slash")
	}
}

func TestSafeRequestID_NonHex(t *testing.T) {
	if SafeRequestID("g1b2c3d4e5f67890") {
		t.Error("expected false for non-hex ID")
	}
}

func TestGenerateRequestID(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id, err := GenerateRequestID()
		if err != nil {
			t.Fatalf("error generating ID: %v", err)
		}
		if len(id) != 16 {
			t.Errorf("expected 16 chars, got %d: %q", len(id), id)
		}
		if !SafeRequestID(id) {
			t.Errorf("generated ID fails safety check: %q", id)
		}
		if seen[id] {
			t.Errorf("duplicate ID generated: %q", id)
		}
		seen[id] = true
	}
}
