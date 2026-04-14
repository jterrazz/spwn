package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

// HelpGroup defines a named group of commands for structured help output.
type HelpGroup struct {
	Title    string
	Commands []HelpEntry
}

// HelpEntry is a single command in a help group.
type HelpEntry struct {
	Name string
	Desc string
}

// ColorizeHelpName applies colors to a command entry name:
//   - flags (starting with -) → yellow
//   - <placeholders> → faint
//   - command words → cyan
func ColorizeHelpName(name string) string {
	return colorizeCmd(name)
}

func colorizeCmd(name string) string {
	parts := strings.Fields(name)
	colored := make([]string, len(parts))
	isFlag := strings.HasPrefix(name, "-")

	for i, p := range parts {
		switch {
		case strings.HasPrefix(p, "<") || strings.HasPrefix(p, "["):
			colored[i] = Faint(p)
		case strings.HasPrefix(p, "-"):
			// Flags are yellow, whether standalone or inline
			colored[i] = Yellow(p)
		case isFlag:
			// Non-placeholder words in a flag entry (e.g. value hint)
			colored[i] = Yellow(p)
		default:
			colored[i] = Cyan(p)
		}
	}
	return strings.Join(colored, " ")
}

// HelpColWidth is the column width used to pad command names in grouped help.
// Entries wider than this still render - they just push past the padding.
const HelpColWidth = 32

// Indent is the per-level indentation used in help output. Headers render
// flush-left; their entries are indented by this much so the eye can skim
// headers and drill into content.
const Indent = "  "

// RenderGroupedHelp writes grouped command help to w.
// Headers sit flush-left, entries are indented one level.
func RenderGroupedHelp(w io.Writer, header string, groups []HelpGroup, usage string, flags string) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s\n", header)
	fmt.Fprintln(w)

	if usage != "" {
		fmt.Fprintf(w, "%s\n", Strong("Usage:"))
		fmt.Fprintf(w, "%s%s\n", Indent, Faint(usage))
		fmt.Fprintln(w)
	}

	for _, g := range groups {
		fmt.Fprintf(w, "%s\n", Strong(g.Title+":"))
		for _, c := range g.Commands {
			if c.Desc == "" {
				fmt.Fprintf(w, "%s%s\n", Indent, colorizeCmd(c.Name))
			} else {
				fmt.Fprintf(w, "%s%s %s\n", Indent, PadVisible(colorizeCmd(c.Name), HelpColWidth), Faint(c.Desc))
			}
		}
		fmt.Fprintln(w)
	}

	if flags != "" {
		fmt.Fprintf(w, "%s\n", Faint(flags))
		fmt.Fprintln(w)
	}
}

// MinimalHelp renders a consistent help view for any cobra command:
// headers flush-left, entries indented one level. Use this as the fallback
// in custom HelpFunc implementations instead of cobra's default (which
// inserts blank lines around every section).
//
// Sections rendered: description, usage, aliases, examples, subcommands,
// local flags, global flags. Sections are omitted when empty.
func MinimalHelp(cmd *cobra.Command, args []string) {
	w := cmd.OutOrStdout()
	fmt.Fprintln(w)

	// Description - prefer Long, fall back to Short. Flush-left.
	if long := strings.TrimSpace(cmd.Long); long != "" {
		fmt.Fprintln(w, long)
		fmt.Fprintln(w)
	} else if short := strings.TrimSpace(cmd.Short); short != "" {
		fmt.Fprintln(w, short)
		fmt.Fprintln(w)
	}

	// Usage.
	if cmd.Runnable() || cmd.HasSubCommands() {
		fmt.Fprintf(w, "%s\n", Strong("Usage:"))
		if cmd.Runnable() {
			fmt.Fprintf(w, "%s%s\n", Indent, Faint(cmd.UseLine()))
		}
		if cmd.HasAvailableSubCommands() {
			fmt.Fprintf(w, "%s%s\n", Indent, Faint(cmd.CommandPath()+" [command]"))
		}
		fmt.Fprintln(w)
	}

	// Aliases.
	if len(cmd.Aliases) > 0 {
		fmt.Fprintf(w, "%s\n", Strong("Aliases:"))
		fmt.Fprintf(w, "%s%s\n", Indent, cmd.NameAndAliases())
		fmt.Fprintln(w)
	}

	// Examples.
	if cmd.HasExample() {
		fmt.Fprintf(w, "%s\n", Strong("Examples:"))
		// Cobra examples are often pre-indented 2 spaces; strip then re-indent
		// consistently so our one-level rule holds.
		for _, line := range strings.Split(strings.TrimRight(cmd.Example, "\n"), "\n") {
			fmt.Fprintf(w, "%s%s\n", Indent, strings.TrimPrefix(line, "  "))
		}
		fmt.Fprintln(w)
	}

	// Subcommands.
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintf(w, "%s\n", Strong("Commands:"))
		for _, c := range cmd.Commands() {
			if !c.IsAvailableCommand() && c.Name() != "help" {
				continue
			}
			fmt.Fprintf(w, "%s%s %s\n", Indent, PadVisible(colorizeCmd(c.Name()), HelpColWidth), Faint(c.Short))
		}
		fmt.Fprintln(w)
	}

	// Local flags - FlagUsages comes pre-indented; re-indent to our width.
	if cmd.HasAvailableLocalFlags() {
		fmt.Fprintf(w, "%s\n", Strong("Flags:"))
		writeIndentedFlagUsages(w, cmd.LocalFlags().FlagUsages())
		fmt.Fprintln(w)
	}

	// Global flags.
	if cmd.HasAvailableInheritedFlags() {
		fmt.Fprintf(w, "%s\n", Strong("Global flags:"))
		writeIndentedFlagUsages(w, cmd.InheritedFlags().FlagUsages())
		fmt.Fprintln(w)
	}
}

// writeIndentedFlagUsages writes FlagUsages output with the pflag-supplied
// 2-space leading whitespace replaced by our standard Indent (also 2
// spaces, but kept routed through the constant for consistency).
func writeIndentedFlagUsages(w io.Writer, usages string) {
	usages = strings.TrimRight(usages, "\n")
	if usages == "" {
		return
	}
	for _, line := range strings.Split(usages, "\n") {
		fmt.Fprintf(w, "%s%s\n", Indent, strings.TrimPrefix(line, "  "))
	}
}
