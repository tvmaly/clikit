package registry

import (
	"testing"

	"github.com/tvmaly/clikit/pkg/cli"
)

func makeCmd(name string) *cli.Command {
	return &cli.Command{Name: name, Short: name + " tool"}
}

func TestRegistry_Register(t *testing.T) {
	r := New()
	r.Register(makeCmd("compliance"))
	cmds := r.Commands()
	if len(cmds) != 1 || cmds[0].Name != "compliance" {
		t.Errorf("expected [compliance], got %v", cmds)
	}
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	r := New()
	r.Register(makeCmd("foo"))
	if err := r.RegisterErr(makeCmd("foo")); err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestRegistry_Lookup_Found(t *testing.T) {
	r := New()
	r.Register(makeCmd("bitbucket"))
	cmd, ok := r.Lookup("bitbucket")
	if !ok {
		t.Fatal("expected to find 'bitbucket'")
	}
	if cmd.Name != "bitbucket" {
		t.Errorf("unexpected name: %q", cmd.Name)
	}
}

func TestRegistry_Lookup_NotFound(t *testing.T) {
	r := New()
	_, ok := r.Lookup("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestRegistry_Commands_Sorted(t *testing.T) {
	r := New()
	r.Register(makeCmd("zebra"))
	r.Register(makeCmd("alpha"))
	r.Register(makeCmd("mango"))
	cmds := r.Commands()
	names := make([]string, len(cmds))
	for i, c := range cmds {
		names[i] = c.Name
	}
	if names[0] != "alpha" || names[1] != "mango" || names[2] != "zebra" {
		t.Errorf("expected sorted [alpha mango zebra], got %v", names)
	}
}

func TestRegistry_Empty(t *testing.T) {
	r := New()
	if len(r.Commands()) != 0 {
		t.Error("expected empty registry")
	}
}
