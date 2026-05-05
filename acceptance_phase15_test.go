package clikit_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhase15AcceptanceArtifactsExist(t *testing.T) {
	required := []string{
		"manifests/schema.json",
		"manifests/codex.json",
		"manifests/hermes.json",
		"manifests/openclaw.json",
		"manifests/nanogo.json",
		"docs/contracts/json-contract-v1.md",
		"docs/runbooks/filequeue-operator-runbook.md",
		"examples/agents/hermes-skill.md",
		"examples/agents/openclaw-skill.md",
		"examples/nanogo/README.md",
		"examples/nanogo/sample-output.json",
	}
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("required Phase 15 artifact %s is missing: %v", path, err)
		}
	}
}

func TestPhase15ManifestsAreMachineReadable(t *testing.T) {
	for _, name := range []string{"codex", "hermes", "openclaw", "nanogo"} {
		path := filepath.Join("manifests", name+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		var manifest struct {
			SchemaVersion string `json:"schema_version"`
			Agent         string `json:"agent"`
			Tool          struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Command     string `json:"command"`
			} `json:"tool"`
			Permissions []string `json:"permissions"`
			Output      struct {
				Default string `json:"default"`
				List    string `json:"list"`
				Errors  string `json:"errors"`
			} `json:"output"`
			RetryBehavior struct {
				Field string `json:"field"`
			} `json:"retry_behavior"`
			Examples []struct {
				Name    string `json:"name"`
				Command string `json:"command"`
				Safety  string `json:"safety"`
			} `json:"examples"`
			SafetyNotes []string `json:"safety_notes"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Fatalf("%s is not valid JSON: %v", path, err)
		}
		if manifest.SchemaVersion != "clikit.manifest/v1" {
			t.Fatalf("%s has schema_version %q", path, manifest.SchemaVersion)
		}
		if manifest.Agent != name {
			t.Fatalf("%s has agent %q, want %q", path, manifest.Agent, name)
		}
		if manifest.Tool.Name == "" || manifest.Tool.Description == "" || manifest.Tool.Command == "" {
			t.Fatalf("%s has incomplete tool metadata", path)
		}
		if len(manifest.Permissions) == 0 || len(manifest.Examples) == 0 || len(manifest.SafetyNotes) == 0 {
			t.Fatalf("%s must define permissions, examples, and safety notes", path)
		}
		if manifest.Output.Default != "json" || manifest.Output.List != "jsonl" || manifest.Output.Errors != "structured-json" {
			t.Fatalf("%s has invalid output contract: %+v", path, manifest.Output)
		}
		if manifest.RetryBehavior.Field != "retry" {
			t.Fatalf("%s must identify retry_behavior.field as retry", path)
		}
		for _, ex := range manifest.Examples {
			if !strings.Contains(ex.Command, "toolkit") {
				t.Fatalf("%s example %q does not call toolkit", path, ex.Name)
			}
			if strings.Contains(ex.Command, " rm ") || strings.Contains(ex.Command, " delete ") {
				t.Fatalf("%s example %q is not read-only: %s", path, ex.Name, ex.Command)
			}
			if !strings.Contains(strings.ToLower(ex.Safety), "read") {
				t.Fatalf("%s example %q must document read-only safety", path, ex.Name)
			}
		}
	}
}

func TestPhase15DocsCoverAcceptanceCriteria(t *testing.T) {
	checks := map[string][]string{
		"docs/contracts/json-contract-v1.md": {
			"Structured Error Contract",
			"Queue Request Metadata",
			"Queue Response Metadata",
			"Command Output Conventions",
			"Breaking Changes",
		},
		"docs/runbooks/filequeue-operator-runbook.md": {
			"Queue Directory Layout",
			"Lifecycle",
			"Secrets",
			"Stale Claim Recovery",
			"Troubleshooting",
		},
		"examples/agents/hermes-skill.md": {
			"allowed-tools",
			"toolkit --help",
			"retry",
			"suggestion",
		},
		"examples/agents/openclaw-skill.md": {
			"allowlist",
			"toolkit --help",
			"retry",
			"suggestion",
		},
		"examples/nanogo/README.md": {
			"read-only",
			"student",
			"compact JSON",
			"not a core dependency",
		},
	}
	for path, needles := range checks {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		text := string(data)
		for _, needle := range needles {
			if !strings.Contains(text, needle) {
				t.Fatalf("%s does not contain %q", path, needle)
			}
		}
	}
}

func TestPhase15NanogoSampleOutputIsCompactJSON(t *testing.T) {
	data, err := os.ReadFile("examples/nanogo/sample-output.json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "\n") > 1 {
		t.Fatalf("sample output must be compact one-line JSON, got %q", data)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("sample output is not JSON: %v", err)
	}
}
