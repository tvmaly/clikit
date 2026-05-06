package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestApp_Run_DispatchesToCommand(t *testing.T) {
	var called string
	app := &App{
		Name: "toolkit",
		Commands: []*Command{
			{
				Name:  "compliance",
				Short: "compliance tools",
				Run: func(ctx *Context) error {
					called = "compliance"
					return nil
				},
			},
		},
	}
	var out bytes.Buffer
	if err := app.Run([]string{"compliance"}, &out, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != "compliance" {
		t.Error("expected compliance command to be called")
	}
}

func TestApp_Run_NoArgs_PrintsHelp(t *testing.T) {
	app := &App{
		Name:     "toolkit",
		Commands: []*Command{{Name: "foo", Short: "bar"}},
	}
	var out bytes.Buffer
	app.Run([]string{}, &out, &out)
	if !strings.Contains(out.String(), "foo") {
		t.Errorf("expected help with 'foo', got %q", out.String())
	}
}

func TestApp_Run_HelpFlag(t *testing.T) {
	app := &App{
		Name:     "toolkit",
		Commands: []*Command{{Name: "foo", Short: "bar"}},
	}
	var out bytes.Buffer
	app.Run([]string{"--help"}, &out, &out)
	if !strings.Contains(out.String(), "toolkit") {
		t.Errorf("expected help output with app name, got %q", out.String())
	}
}

func TestApp_Run_UnknownCommand_Error(t *testing.T) {
	app := &App{
		Name:     "toolkit",
		Commands: []*Command{{Name: "known", Short: "k"}},
	}
	var out bytes.Buffer
	err := app.Run([]string{"unknown"}, &out, &out)
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestApp_Run_GlobalFlags_BeforeCommand(t *testing.T) {
	var gotPretty bool
	app := &App{
		Name: "toolkit",
		Commands: []*Command{
			{
				Name:  "cmd",
				Short: "c",
				Run: func(ctx *Context) error {
					gotPretty = ctx.GlobalOpts.Pretty
					return nil
				},
			},
		},
	}
	var out bytes.Buffer
	app.Run([]string{"--pretty", "cmd"}, &out, &out)
	if !gotPretty {
		t.Error("expected pretty=true to be passed to command context")
	}
}

func TestApp_Run_PresentsCommandOutputForInjectedWriter(t *testing.T) {
	app := &App{
		Name: "toolkit",
		Commands: []*Command{
			{
				Name: "cmd",
				Run: func(ctx *Context) error {
					_, err := ctx.Stdout.Write([]byte(`{"ok":true}` + "\n"))
					return err
				},
			},
		},
	}
	var out bytes.Buffer
	if err := app.Run([]string{"cmd"}, &out, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, `{"ok":true}`) {
		t.Fatalf("expected command output, got %q", got)
	}
	if !strings.Contains(got, "[exit:0 |") {
		t.Fatalf("expected presentation footer, got %q", got)
	}
}

func TestApp_Run_RawSkipsPresentation(t *testing.T) {
	app := &App{
		Name: "toolkit",
		Commands: []*Command{
			{
				Name: "cmd",
				Run: func(ctx *Context) error {
					_, err := ctx.Stdout.Write([]byte(`{"ok":true}` + "\n"))
					return err
				},
			},
		},
	}
	var out bytes.Buffer
	if err := app.Run([]string{"--raw", "cmd"}, &out, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out.String(), "[exit:0 |") {
		t.Fatalf("raw output must not include presentation footer, got %q", out.String())
	}
}

func TestApp_Run_FromStdinRejectsBinary(t *testing.T) {
	app := &App{
		Name: "toolkit",
		Commands: []*Command{
			{
				Name: "cmd",
				Run: func(ctx *Context) error {
					t.Fatal("command should not run with binary stdin")
					return nil
				},
			},
		},
		Stdin: bytes.NewReader([]byte{0x00, 0x01, 0x02}),
	}
	var out bytes.Buffer
	err := app.Run([]string{"--from-stdin", "cmd"}, &out, &out)
	if err == nil {
		t.Fatal("expected binary stdin error")
	}
	if !strings.Contains(err.Error(), "binary stdin") {
		t.Fatalf("expected binary stdin error, got %v", err)
	}
	if !strings.Contains(out.String(), `"error"`) || !strings.Contains(out.String(), `"suggestion"`) {
		t.Fatalf("expected structured binary stdin error on stdout, got %q", out.String())
	}
}

func TestApp_Run_GlobalOutputFlags_FieldQuietPrettyJSONL(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		payload  string
		contains []string
	}{
		{
			name:     "field",
			args:     []string{"--field", "user.name", "cmd"},
			payload:  `{"user":{"name":"ada"},"id":1}` + "\n",
			contains: []string{"ada", "[exit:0 |"},
		},
		{
			name:     "quiet field",
			args:     []string{"--quiet", "--field", "user.name", "cmd"},
			payload:  `{"user":{"name":"ada"},"id":1}` + "\n",
			contains: []string{"ada", "[exit:0 |"},
		},
		{
			name:     "pretty",
			args:     []string{"--pretty", "cmd"},
			payload:  `{"ok":true,"name":"ada"}` + "\n",
			contains: []string{"\n  \"ok\": true", "[exit:0 |"},
		},
		{
			name:     "jsonl",
			args:     []string{"--output", "jsonl", "cmd"},
			payload:  `[{"id":1},{"id":2}]` + "\n",
			contains: []string{`{"id":1}` + "\n" + `{"id":2}`, "[exit:0 |"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := appWithPayload(tt.payload)
			var out bytes.Buffer
			if err := app.Run(tt.args, &out, &out); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, want := range tt.contains {
				if !strings.Contains(out.String(), want) {
					t.Fatalf("expected %q in output, got %q", want, out.String())
				}
			}
		})
	}
}

func TestApp_Run_PipedStdoutSkipsPresentation(t *testing.T) {
	app := appWithPayload(`{"ok":true}` + "\n")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := app.Run([]string{"cmd"}, w, io.Discard); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if strings.Contains(got, "[exit:0 |") {
		t.Fatalf("piped output must not include presentation footer, got %q", got)
	}
	if got != `{"ok":true}`+"\n" {
		t.Fatalf("unexpected piped output: %q", got)
	}
}

func TestApp_Run_OverflowAndBinaryOutputAtAppBoundary(t *testing.T) {
	t.Run("overflow", func(t *testing.T) {
		app := appWithPayload(`{"data":"` + strings.Repeat("x", 100) + `"}` + "\n")
		app.OverflowThreshold = 20
		app.SpillDir = t.TempDir()
		var out bytes.Buffer
		if err := app.Run([]string{"cmd"}, &out, &out); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(out.String(), "truncated") {
			t.Fatalf("expected truncation message, got %q", out.String())
		}
	})

	t.Run("binary output", func(t *testing.T) {
		app := appWithPayload("abc\x00def")
		var out bytes.Buffer
		err := app.Run([]string{"cmd"}, &out, &out)
		if err == nil || !strings.Contains(err.Error(), "binary output") {
			t.Fatalf("expected binary output error, got %v", err)
		}
	})
}

func TestApp_Run_BinaryStdinStructuredErrorIsValidJSON(t *testing.T) {
	app := appWithPayload(`{"unused":true}`)
	app.Stdin = bytes.NewReader([]byte{0x00, 0x01})
	var out bytes.Buffer
	err := app.Run([]string{"--from-stdin", "cmd"}, &out, &out)
	if err == nil {
		t.Fatal("expected error")
	}
	var payload map[string]any
	if jsonErr := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &payload); jsonErr != nil {
		t.Fatalf("expected JSON error payload, got %q: %v", out.String(), jsonErr)
	}
	if payload["retry"] != false {
		t.Fatalf("expected retry=false, got %#v", payload)
	}
}

func appWithPayload(payload string) *App {
	return &App{
		Name: "toolkit",
		Commands: []*Command{
			{
				Name: "cmd",
				Run: func(ctx *Context) error {
					_, err := ctx.Stdout.Write([]byte(payload))
					return err
				},
			},
		},
	}
}
