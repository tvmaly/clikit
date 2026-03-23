package transport

import (
	"testing"
)

func TestNewTransport_HTTP(t *testing.T) {
	cfg := &Config{Mode: "http", APIURL: "http://localhost:8080", APIToken: "tok"}
	tr, err := NewTransport(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr == nil {
		t.Error("expected non-nil transport")
	}
	tr.Close()
}

func TestNewTransport_FileQueue(t *testing.T) {
	cfg := &Config{Mode: "filequeue", QueueRoot: t.TempDir()}
	tr, err := NewTransport(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr == nil {
		t.Error("expected non-nil transport")
	}
	tr.Close()
}

func TestNewTransport_Unknown(t *testing.T) {
	cfg := &Config{Mode: "invalid"}
	_, err := NewTransport(cfg)
	if err == nil {
		t.Error("expected error for unknown transport mode")
	}
}
