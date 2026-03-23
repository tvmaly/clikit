package cli

import (
	"bytes"
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
