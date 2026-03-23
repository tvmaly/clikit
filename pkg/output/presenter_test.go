package output

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestPresenter_PlainOutput(t *testing.T) {
	var out bytes.Buffer
	p := NewPresenter(&out, Options{})
	data := []byte(`{"key":"value"}`)
	if err := p.Present(data, 0, nil, time.Millisecond*50); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "key") {
		t.Errorf("expected key in output, got %q", out.String())
	}
}

func TestPresenter_MetadataFooter_Milliseconds(t *testing.T) {
	var out bytes.Buffer
	p := NewPresenter(&out, Options{})
	data := []byte(`{"k":"v"}`)
	dur := 450 * time.Millisecond
	if err := p.Present(data, 0, nil, dur); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	footer := out.String()
	if !strings.Contains(footer, "[exit:0 |") {
		t.Errorf("expected metadata footer with exit:0, got %q", footer)
	}
	if !strings.Contains(footer, "ms]") {
		t.Errorf("expected ms in footer, got %q", footer)
	}
}

func TestPresenter_MetadataFooter_Seconds(t *testing.T) {
	var out bytes.Buffer
	p := NewPresenter(&out, Options{})
	data := []byte(`{"k":"v"}`)
	dur := 1500 * time.Millisecond
	if err := p.Present(data, 0, nil, dur); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	footer := out.String()
	if !strings.Contains(footer, "s]") {
		t.Errorf("expected seconds in footer, got %q", footer)
	}
}

func TestPresenter_NonZeroExit_AttachesStderr(t *testing.T) {
	var out bytes.Buffer
	p := NewPresenter(&out, Options{})
	data := []byte(`{"error":"bad"}`)
	stderr := []byte("something went wrong\n")
	if err := p.Present(data, 1, stderr, time.Millisecond*10); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := out.String()
	if !strings.Contains(result, "[exit:1 |") {
		t.Errorf("expected exit:1 in footer, got %q", result)
	}
	if !strings.Contains(result, "something went wrong") {
		t.Errorf("expected stderr in output, got %q", result)
	}
}

func TestPresenter_BinaryGuard(t *testing.T) {
	var out bytes.Buffer
	p := NewPresenter(&out, Options{})
	// Data with null byte — binary
	data := []byte("PNG\x00\x01\x02\x03binary content")
	err := p.Present(data, 0, nil, time.Millisecond*5)
	if err == nil {
		t.Error("expected error for binary output")
	}
	if !strings.Contains(err.Error(), "binary") {
		t.Errorf("expected 'binary' in error, got %q", err.Error())
	}
}

func TestPresenter_Overflow(t *testing.T) {
	var out bytes.Buffer
	spillDir := t.TempDir()
	p := NewPresenter(&out, Options{
		OverflowThreshold: 20,
		SpillDir:          spillDir,
	})
	data := []byte(`{"key":"` + strings.Repeat("x", 100) + `"}`)
	if err := p.Present(data, 0, nil, time.Millisecond*5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := out.String()
	if !strings.Contains(result, "truncated") {
		t.Errorf("expected overflow hint in output, got %q", result)
	}
}

func TestPresenter_ZeroExitNoStderr(t *testing.T) {
	var out bytes.Buffer
	p := NewPresenter(&out, Options{})
	data := []byte(`{"ok":true}`)
	stderr := []byte("some warning")
	if err := p.Present(data, 0, stderr, time.Millisecond); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// stderr should NOT appear on exit 0
	if strings.Contains(out.String(), "some warning") {
		t.Errorf("expected stderr suppressed on exit 0, got %q", out.String())
	}
}
