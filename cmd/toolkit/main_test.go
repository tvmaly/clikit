package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tvmaly/clikit/pkg/filequeue"
	"github.com/tvmaly/clikit/pkg/tooldefs"
)

func TestRunHelpListsBuiltInTools(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%q", code, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{"example", "queue"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in help, got %q", want, got)
		}
	}
}

func TestRunExampleGetOutputsCompactJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--raw", "example", "get", "abc"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got["id"] != "abc" || got["name"] != "example-item" {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestExampleOperationRawInvokeMatchesToolkitCommand(t *testing.T) {
	op := tooldefs.ExampleOperations()[0]
	result, err := op.Call(context.Background(), json.RawMessage(`{"id":"abc"}`))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"--raw", "example", "get", "--id", "abc"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if strings.TrimSpace(stdout.String()) != string(result.Body) {
		t.Fatalf("raw operation and CLI output differ: op=%q cli=%q", result.Body, stdout.String())
	}
}

func TestRunQueueStatusListInspectRetryClean(t *testing.T) {
	root := t.TempDir()
	if err := filequeue.EnsureQueueDirs(root); err != nil {
		t.Fatal(err)
	}
	p := filequeue.QueuePaths(root)
	id := "a1b2c3d4e5f67890"
	meta := &filequeue.RequestMeta{
		RequestID: id,
		Tool:      "example",
		Operation: "get",
		CreatedAt: time.Now().UTC(),
	}
	data, _ := json.Marshal(meta)
	if err := filequeue.AtomicWrite(p.Pending, id+".meta.json", data); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"--raw", "queue", "status", "--queue-root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("status exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"pending":1`) {
		t.Fatalf("expected pending count, got %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--raw", "queue", "list", "--queue-root", root, "--state", "pending"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("list exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"request_id":"`+id+`"`) {
		t.Fatalf("expected request id in list, got %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--raw", "queue", "inspect", "--queue-root", root, id}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("inspect exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"state":"pending"`) {
		t.Fatalf("expected state in inspect, got %q", stdout.String())
	}

	// Move the request to dead, then retry it to pending.
	if err := os.Rename(filepath.Join(p.Pending, id+".meta.json"), filepath.Join(p.Dead, id+".meta.json")); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--raw", "queue", "retry", "--queue-root", root, id}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("retry exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if _, err := os.Stat(filepath.Join(p.Pending, id+".meta.json")); err != nil {
		t.Fatalf("expected request moved back to pending: %v", err)
	}

	oldResp := &filequeue.ResponseMeta{RequestID: id, CompletedAt: time.Now().Add(-48 * time.Hour)}
	respData, _ := json.Marshal(oldResp)
	if err := filequeue.AtomicWrite(p.Done, id+".meta.json", respData); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--raw", "queue", "clean", "--queue-root", root, "--done-ttl", "1h"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("clean exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"cleaned":1`) {
		t.Fatalf("expected cleaned count, got %q", stdout.String())
	}
}

func TestRunSkillGenOutputsSkillMarkdown(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--raw", "skill-gen"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	got := stdout.String()
	for _, want := range []string{
		"---",
		"name: toolkit",
		"allowed-tools: [Bash(toolkit *), Bash(toolkit-fmt *)]",
		"## Output conventions",
		"## Error handling",
		"## Queue operations",
		"Never guess flags",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in generated skill, got %q", want, got)
		}
	}
}
