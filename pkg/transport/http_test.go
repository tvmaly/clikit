package transport

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPTransport_Execute_GET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	cfg := &Config{Mode: "http", APIURL: srv.URL, RequestTimeout: 5 * time.Second}
	tr, _ := NewHTTPTransport(cfg)
	defer tr.Close()

	resp, err := tr.Execute(context.Background(), &Request{Method: "GET", Path: "/api/v1/test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(resp.Body), "ok") {
		t.Errorf("unexpected body: %q", resp.Body)
	}
}

func TestHTTPTransport_Execute_POST(t *testing.T) {
	var receivedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"echo":true}`))
	}))
	defer srv.Close()

	cfg := &Config{Mode: "http", APIURL: srv.URL, RequestTimeout: 5 * time.Second}
	tr, _ := NewHTTPTransport(cfg)
	defer tr.Close()

	body := []byte(`{"name":"test"}`)
	resp, err := tr.Execute(context.Background(), &Request{Method: "POST", Path: "/api/v1/test", Body: body})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if string(receivedBody) != string(body) {
		t.Errorf("expected server to receive %q, got %q", body, receivedBody)
	}
}

func TestHTTPTransport_Execute_AuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	cfg := &Config{Mode: "http", APIURL: srv.URL, APIToken: "test-token", RequestTimeout: 5 * time.Second}
	tr, _ := NewHTTPTransport(cfg)
	defer tr.Close()

	tr.Execute(context.Background(), &Request{Method: "GET", Path: "/"})
	if gotAuth != "Bearer test-token" {
		t.Errorf("expected 'Bearer test-token', got %q", gotAuth)
	}
}

func TestHTTPTransport_Execute_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay longer than client timeout.
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer srv.Close()

	cfg := &Config{Mode: "http", APIURL: srv.URL, RequestTimeout: 100 * time.Millisecond}
	tr, _ := NewHTTPTransport(cfg)
	defer tr.Close()

	resp, err := tr.Execute(context.Background(), &Request{Method: "GET", Path: "/"})
	if err == nil && resp.Error == "" {
		t.Error("expected error or non-empty Response.Error for timeout")
	}
}

func TestHTTPTransport_Execute_NetworkError(t *testing.T) {
	cfg := &Config{Mode: "http", APIURL: "http://127.0.0.1:1", RequestTimeout: 500 * time.Millisecond}
	tr, _ := NewHTTPTransport(cfg)
	defer tr.Close()

	resp, err := tr.Execute(context.Background(), &Request{Method: "GET", Path: "/"})
	// Should return gracefully with either an error or populated Response.Error.
	if err == nil && (resp == nil || resp.Error == "") {
		t.Error("expected error for network failure")
	}
}

func TestHTTPTransport_Execute_QueryParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	cfg := &Config{Mode: "http", APIURL: srv.URL, RequestTimeout: 5 * time.Second}
	tr, _ := NewHTTPTransport(cfg)
	defer tr.Close()

	tr.Execute(context.Background(), &Request{
		Method: "GET",
		Path:   "/api",
		Params: map[string]string{"key": "val", "page": "2"},
	})
	if !strings.Contains(gotQuery, "key=val") || !strings.Contains(gotQuery, "page=2") {
		t.Errorf("expected query params, got %q", gotQuery)
	}
}
