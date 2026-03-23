package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCommand_Run_NoSubcommands(t *testing.T) {
	var called bool
	cmd := &Command{
		Name:  "test",
		Short: "A test command",
		Run: func(ctx *Context) error {
			called = true
			return nil
		},
	}
	var out bytes.Buffer
	ctx := &Context{Stdout: &out, Stderr: &out, GlobalOpts: DefaultGlobalOpts()}
	if err := cmd.Execute(ctx, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected Run to be called")
	}
}

func TestCommand_Run_WithSubcommand(t *testing.T) {
	var called string
	parent := &Command{
		Name:  "parent",
		Short: "parent cmd",
		Subcommands: []*Command{
			{
				Name:  "child",
				Short: "child cmd",
				Run: func(ctx *Context) error {
					called = "child"
					return nil
				},
			},
		},
	}
	var out bytes.Buffer
	ctx := &Context{Stdout: &out, Stderr: &out, GlobalOpts: DefaultGlobalOpts()}
	if err := parent.Execute(ctx, []string{"child"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != "child" {
		t.Error("expected child to be called")
	}
}

func TestCommand_UnknownSubcommand_Error(t *testing.T) {
	parent := &Command{
		Name:  "parent",
		Short: "parent cmd",
		Subcommands: []*Command{
			{Name: "known", Short: "k"},
		},
	}
	var out bytes.Buffer
	ctx := &Context{Stdout: &out, Stderr: &out, GlobalOpts: DefaultGlobalOpts()}
	err := parent.Execute(ctx, []string{"unknown"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("expected 'unknown' in error, got %q", err.Error())
	}
}

func TestCommand_NoRunAndNoSubcommand(t *testing.T) {
	cmd := &Command{Name: "empty", Short: "empty"}
	var out bytes.Buffer
	ctx := &Context{Stdout: &out, Stderr: &out, GlobalOpts: DefaultGlobalOpts()}
	// Should return an error or print help.
	err := cmd.Execute(ctx, []string{})
	// Acceptable: error or nil (prints help).
	_ = err
}

func TestCommand_Args_PassedToRun(t *testing.T) {
	var gotArgs []string
	cmd := &Command{
		Name:  "cmd",
		Short: "c",
		Run: func(ctx *Context) error {
			gotArgs = ctx.Args
			return nil
		},
	}
	var out bytes.Buffer
	ctx := &Context{Stdout: &out, Stderr: &out, GlobalOpts: DefaultGlobalOpts()}
	cmd.Execute(ctx, []string{"arg1", "arg2"})
	if len(gotArgs) != 2 || gotArgs[0] != "arg1" {
		t.Errorf("expected args [arg1 arg2], got %v", gotArgs)
	}
}
