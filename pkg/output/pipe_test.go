package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestIsPiped_WriterIsNotFile(t *testing.T) {
	// A bytes.Buffer is not a file, so IsPipedWriter should return true
	// (treat non-*os.File writers as piped).
	var buf bytes.Buffer
	if !IsPipedWriter(&buf) {
		t.Error("expected true for non-file writer")
	}
}

func TestCopyRaw(t *testing.T) {
	src := strings.NewReader("hello raw output")
	var dst bytes.Buffer
	if err := CopyRaw(&dst, src); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dst.String() != "hello raw output" {
		t.Errorf("expected 'hello raw output', got %q", dst.String())
	}
}

func TestCopyRaw_Empty(t *testing.T) {
	src := strings.NewReader("")
	var dst bytes.Buffer
	if err := CopyRaw(&dst, src); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dst.String() != "" {
		t.Errorf("expected empty output, got %q", dst.String())
	}
}

func TestCopyRaw_LargeData(t *testing.T) {
	large := strings.Repeat("abcdefgh", 10000) // 80KB
	src := strings.NewReader(large)
	var dst bytes.Buffer
	if err := CopyRaw(&dst, src); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dst.Len() != len(large) {
		t.Errorf("expected %d bytes, got %d", len(large), dst.Len())
	}
}
