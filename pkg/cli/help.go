package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
)

// RenderRegistryHelp writes Level 0 help: list of all registered tools.
func RenderRegistryHelp(w io.Writer, appName string, cmds []*Command) {
	fmt.Fprintf(w, "Usage: %s <tool> [subcommand] [flags]\n\n", appName)
	fmt.Fprintln(w, "Available tools:")

	sorted := make([]*Command, len(cmds))
	copy(sorted, cmds)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, c := range sorted {
		fmt.Fprintf(tw, "  %s\t%s\n", c.Name, c.Short)
	}
	tw.Flush()
	fmt.Fprintf(w, "\nRun '%s <tool>' for subcommands.\n", appName)
}

// RenderCommandHelp writes Level 1 help: list of subcommands for a tool.
func RenderCommandHelp(w io.Writer, cmd *Command) {
	desc := cmd.Short
	if cmd.Long != "" {
		desc = cmd.Long
	}
	fmt.Fprintf(w, "%s: %s\n\n", cmd.Name, desc)

	if len(cmd.Subcommands) > 0 {
		fmt.Fprintln(w, "Available subcommands:")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, sub := range cmd.Subcommands {
			fmt.Fprintf(tw, "  %s\t%s\n", sub.Name, sub.Short)
		}
		tw.Flush()
	}
}

// RenderSubcommandHelp writes Level 2 help: details for a specific subcommand.
func RenderSubcommandHelp(w io.Writer, cmd *Command) {
	fmt.Fprintf(w, "Usage: %s [flags]\n\n", cmd.Name)
	desc := cmd.Short
	if cmd.Long != "" {
		desc = cmd.Long
	}
	fmt.Fprintln(w, strings.TrimSpace(desc))
}
