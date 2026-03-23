package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelp_Level0_Registry(t *testing.T) {
	cmds := []*Command{
		{Name: "compliance", Short: "Compliance scanning tools"},
		{Name: "bitbucket", Short: "Bitbucket repository tools"},
	}
	var buf bytes.Buffer
	RenderRegistryHelp(&buf, "toolkit", cmds)
	out := buf.String()
	if !strings.Contains(out, "compliance") {
		t.Errorf("expected 'compliance' in registry help, got %q", out)
	}
	if !strings.Contains(out, "bitbucket") {
		t.Errorf("expected 'bitbucket' in registry help, got %q", out)
	}
}

func TestHelp_Level1_Command(t *testing.T) {
	cmd := &Command{
		Name:  "compliance",
		Short: "Compliance scanning tools",
		Long:  "Longer description of compliance.",
		Subcommands: []*Command{
			{Name: "scan", Short: "Run a compliance scan"},
			{Name: "rules", Short: "List compliance rules"},
		},
	}
	var buf bytes.Buffer
	RenderCommandHelp(&buf, cmd)
	out := buf.String()
	if !strings.Contains(out, "scan") {
		t.Errorf("expected 'scan' in command help, got %q", out)
	}
	if !strings.Contains(out, "rules") {
		t.Errorf("expected 'rules' in command help, got %q", out)
	}
}

func TestHelp_Level2_Subcommand(t *testing.T) {
	sub := &Command{
		Name:  "scan",
		Short: "Run a compliance scan",
		Long:  "Detailed scan instructions.",
	}
	var buf bytes.Buffer
	RenderSubcommandHelp(&buf, sub)
	out := buf.String()
	if !strings.Contains(out, "scan") {
		t.Errorf("expected subcommand name in help, got %q", out)
	}
	if !strings.Contains(out, "Detailed") {
		t.Errorf("expected long description in help, got %q", out)
	}
}
