package worker

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleHandlerConfig = `{
  "handlers": [
    {
      "tool": "compliance",
      "operation": "scan",
      "method": "GET",
      "url_path": "/api/v1/compliance/scan",
      "base_url": "http://compliance-svc.internal:8080",
      "token_env": "COMPLIANCE_API_TOKEN",
      "timeout_seconds": 60
    },
    {
      "tool": "bitbucket",
      "operation": "repo-list",
      "method": "GET",
      "url_path": "/rest/api/1.0/projects/{project}/repos",
      "base_url": "https://bitbucket.internal",
      "token_env": "BITBUCKET_TOKEN",
      "timeout_seconds": 30
    }
  ]
}`

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "handler-config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()
	return f.Name()
}

func TestLoadHandlerConfig_Valid(t *testing.T) {
	path := writeTempConfig(t, sampleHandlerConfig)
	cfg, err := LoadHandlerConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Handlers) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(cfg.Handlers))
	}
}

func TestLoadHandlerConfig_InvalidJSON(t *testing.T) {
	path := writeTempConfig(t, "not json")
	_, err := LoadHandlerConfig(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadHandlerConfig_MissingFields(t *testing.T) {
	path := writeTempConfig(t, `{"handlers":[{"tool":"","operation":""}]}`)
	_, err := LoadHandlerConfig(path)
	if err == nil {
		t.Error("expected validation error for missing tool/operation")
	}
}

func TestLookup_Found(t *testing.T) {
	path := writeTempConfig(t, sampleHandlerConfig)
	cfg, _ := LoadHandlerConfig(path)
	entry, ok := cfg.Lookup("compliance", "scan")
	if !ok {
		t.Fatal("expected to find handler")
	}
	if entry.URLPath != "/api/v1/compliance/scan" {
		t.Errorf("unexpected url_path: %q", entry.URLPath)
	}
}

func TestLookup_NotFound(t *testing.T) {
	path := writeTempConfig(t, sampleHandlerConfig)
	cfg, _ := LoadHandlerConfig(path)
	_, ok := cfg.Lookup("nonexistent", "op")
	if ok {
		t.Error("expected not found")
	}
}

func TestURLTemplateSubstitution(t *testing.T) {
	entry := &HandlerEntry{
		URLPath: "/rest/api/1.0/projects/{project}/repos",
		BaseURL: "https://bitbucket.internal",
	}
	params := map[string]string{"project": "MYPROJ", "limit": "25"}
	finalURL, remainingParams, err := BuildURL(entry, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if finalURL != "https://bitbucket.internal/rest/api/1.0/projects/MYPROJ/repos" {
		t.Errorf("unexpected URL: %q", finalURL)
	}
	if remainingParams["limit"] != "25" {
		t.Errorf("expected limit=25 in remaining params")
	}
	if _, ok := remainingParams["project"]; ok {
		t.Error("project should be consumed from params")
	}
}

func TestTokenEnvResolution(t *testing.T) {
	t.Setenv("COMPLIANCE_API_TOKEN", "secret123")
	path := writeTempConfig(t, sampleHandlerConfig)
	cfg, err := LoadHandlerConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entry, _ := cfg.Lookup("compliance", "scan")
	if entry.Token != "secret123" {
		t.Errorf("expected token 'secret123', got %q", entry.Token)
	}
	_ = filepath.Join(".")
}
