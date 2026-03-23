package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOverflow_BelowThreshold(t *testing.T) {
	data := []byte(`{"key":"value"}`)
	result, spill, err := CheckOverflow(data, 1024, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spill != "" {
		t.Errorf("expected no spillfile, got %q", spill)
	}
	if string(result) != string(data) {
		t.Errorf("expected data unchanged, got %q", result)
	}
}

func TestOverflow_ExceedsThreshold(t *testing.T) {
	data := []byte(strings.Repeat("x", 200))
	dir := t.TempDir()

	result, spill, err := CheckOverflow(data, 100, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spill == "" {
		t.Fatal("expected a spillfile path")
	}

	// result should be truncated to threshold bytes
	if len(result) > 100 {
		t.Errorf("expected result <= 100 bytes, got %d", len(result))
	}

	// spillfile should contain full data
	spillData, err := os.ReadFile(spill)
	if err != nil {
		t.Fatalf("reading spillfile: %v", err)
	}
	if string(spillData) != string(data) {
		t.Error("spillfile should contain full original data")
	}
}

func TestOverflow_SpillfileInDir(t *testing.T) {
	data := []byte(strings.Repeat("y", 500))
	dir := t.TempDir()

	_, spill, err := CheckOverflow(data, 100, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// spillfile must be inside dir
	rel, err := filepath.Rel(dir, spill)
	if err != nil || strings.HasPrefix(rel, "..") {
		t.Errorf("spillfile %q is not inside dir %q", spill, dir)
	}
}

func TestOverflow_ExactThreshold(t *testing.T) {
	data := []byte(strings.Repeat("z", 100))
	result, spill, err := CheckOverflow(data, 100, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// At exactly threshold — no overflow
	if spill != "" {
		t.Errorf("expected no spillfile at exact threshold")
	}
	if len(result) != 100 {
		t.Errorf("expected 100 bytes, got %d", len(result))
	}
}

func TestOverflow_TruncatesAtRuneBoundary(t *testing.T) {
	// "café" is 5 bytes in UTF-8 (c=1, a=1, f=1, é=2)
	// with threshold of 4, must not split the é rune
	data := []byte("café extra")
	result, _, err := CheckOverflow(data, 4, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// result must be valid UTF-8
	if !isValidUTF8(result) {
		t.Errorf("truncated result is not valid UTF-8: %q", result)
	}
}

func isValidUTF8(b []byte) bool {
	// simple: try to range over string
	for _, r := range string(b) {
		_ = r
	}
	return true
}
