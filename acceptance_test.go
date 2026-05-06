package clikit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tvmaly/clikit/pkg/filequeue"
	"github.com/tvmaly/clikit/pkg/transport"
	"github.com/tvmaly/clikit/pkg/worker"
)

func TestAcceptance_TransportTransparentHTTPAndFileQueue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/example" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if r.URL.Query().Get("id") != "abc" {
			t.Fatalf("unexpected id query %q", r.URL.Query().Get("id"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"abc","name":"example-item"}`))
	}))
	defer srv.Close()

	req := &transport.Request{
		Tool:      "example",
		Operation: "get",
		Method:    "GET",
		Path:      "/api/example",
		Params:    map[string]string{"id": "abc"},
	}

	httpTransport, err := transport.NewHTTPTransport(&transport.Config{
		APIURL:         srv.URL,
		RequestTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	httpResp, err := httpTransport.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	queueRoot := t.TempDir()
	if err := filequeue.EnsureQueueDirs(queueRoot); err != nil {
		t.Fatal(err)
	}
	fileQueueTransport, err := transport.NewFileQueueTransport(&transport.Config{
		QueueRoot:      queueRoot,
		PollInterval:   10 * time.Millisecond,
		RequestTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := &worker.Dispatcher{
		QueueRoot:    queueRoot,
		WorkerID:     "acceptance-worker",
		Handlers:     &worker.HandlerConfig{Handlers: []worker.HandlerEntry{{Tool: "example", Operation: "get", Method: "GET", BaseURL: srv.URL, URLPath: "/api/example", Timeout: 5}}},
		HTTPClient:   srv.Client(),
		StaleClaim:   time.Minute,
		PollInterval: 10 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go dispatcher.Run(ctx) //nolint:errcheck

	fileResp, err := fileQueueTransport.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if httpResp.StatusCode != fileResp.StatusCode {
		t.Fatalf("status mismatch: http=%d filequeue=%d", httpResp.StatusCode, fileResp.StatusCode)
	}
	if string(httpResp.Body) != string(fileResp.Body) {
		t.Fatalf("body mismatch: http=%q filequeue=%q", httpResp.Body, fileResp.Body)
	}
}
