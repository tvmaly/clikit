package cli

import (
	"testing"
)

func TestGlobalOpts_Defaults(t *testing.T) {
	opts := DefaultGlobalOpts()
	if opts.Output != "json" {
		t.Errorf("expected default output='json', got %q", opts.Output)
	}
	if opts.Pretty {
		t.Error("expected pretty=false by default")
	}
	if opts.Quiet {
		t.Error("expected quiet=false by default")
	}
	if opts.Raw {
		t.Error("expected raw=false by default")
	}
}

func TestParseGlobalFlags_Output(t *testing.T) {
	opts, remaining, err := ParseGlobalFlags([]string{"--output", "jsonl", "subcmd", "--flag"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Output != "jsonl" {
		t.Errorf("expected output='jsonl', got %q", opts.Output)
	}
	if len(remaining) != 2 || remaining[0] != "subcmd" {
		t.Errorf("unexpected remaining: %v", remaining)
	}
}

func TestParseGlobalFlags_Pretty(t *testing.T) {
	opts, _, err := ParseGlobalFlags([]string{"--pretty"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Pretty {
		t.Error("expected pretty=true")
	}
}

func TestParseGlobalFlags_Quiet(t *testing.T) {
	opts, _, err := ParseGlobalFlags([]string{"--quiet", "--field", "name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Quiet {
		t.Error("expected quiet=true")
	}
	if opts.Field != "name" {
		t.Errorf("expected field='name', got %q", opts.Field)
	}
}

func TestParseGlobalFlags_Raw(t *testing.T) {
	opts, _, err := ParseGlobalFlags([]string{"--raw"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Raw {
		t.Error("expected raw=true")
	}
}

func TestParseGlobalFlags_Limit(t *testing.T) {
	opts, _, err := ParseGlobalFlags([]string{"--limit", "50"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Limit != 50 {
		t.Errorf("expected limit=50, got %d", opts.Limit)
	}
}

func TestParseGlobalFlags_UnknownFlagPassThrough(t *testing.T) {
	// Unknown flags should be passed through as remaining args.
	_, remaining, _ := ParseGlobalFlags([]string{"--pretty", "command", "--custom-flag", "val"})
	found := false
	for _, r := range remaining {
		if r == "--custom-flag" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected --custom-flag in remaining: %v", remaining)
	}
}
