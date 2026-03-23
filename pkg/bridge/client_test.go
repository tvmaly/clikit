package bridge

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClient_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"result":"ok"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", 5*time.Second)
	body, status, err := c.Do(context.Background(), "GET", "/test", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 200 {
		t.Errorf("expected 200, got %d", status)
	}
	if !strings.Contains(string(body), "ok") {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestClient_Post(t *testing.T) {
	var got []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"echo":true}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", 5*time.Second)
	payload := []byte(`{"name":"test"}`)
	_, _, err := c.Do(context.Background(), "POST", "/test", payload, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("server received %q, expected %q", got, payload)
	}
}

func TestClient_AuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "mytoken", 5*time.Second)
	c.Do(context.Background(), "GET", "/", nil, nil)
	if gotAuth != "Bearer mytoken" {
		t.Errorf("expected 'Bearer mytoken', got %q", gotAuth)
	}
}

func TestClient_QueryParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", 5*time.Second)
	c.Do(context.Background(), "GET", "/api", nil, map[string]string{"page": "2", "limit": "10"})
	if !strings.Contains(gotQuery, "page=2") || !strings.Contains(gotQuery, "limit=10") {
		t.Errorf("expected query params, got %q", gotQuery)
	}
}

func TestClient_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "", 100*time.Millisecond)
	_, _, err := c.Do(context.Background(), "GET", "/", nil, nil)
	if err == nil {
		t.Error("expected error for network failure")
	}
}
