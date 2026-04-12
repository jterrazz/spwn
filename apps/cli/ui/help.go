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
// Entries wider than this still render — they just push past the padding.
const HelpColWidth = 32

// RenderGroupedHelp writes grouped command help to w. Flush-left, no padding.
func RenderGroupedHelp(w io.Writer, header string, groups []HelpGroup, usage string, flags string) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s\n", header)
	fmt.Fprintln(w)

	if usage != "" {
		fmt.Fprintf(w, "%s\n", Strong("Usage:"))
		fmt.Fprintf(w, "%s\n", Faint(usage))
		fmt.Fprintln(w)
	}

	for _, g := range groups {
		fmt.Fprintf(w, "%s\n", Strong(g.Title+":"))
		for _, c := range g.Commands {
			if c.Desc == "" {
				fmt.Fprintf(w, "%s\n", colorizeCmd(c.Name))
			} else {
				fmt.Fprintf(w, "%s %s\n", PadVisible(colorizeCmd(c.Name), HelpColWidth), Faint(c.Desc))
			}
		}
		fmt.Fprintln(w)
	}

	if flags != "" {
		fmt.Fprintf(w, "%s\n", Faint(flags))
		fmt.Fprintln(w)
	}
}

// MinimalHelp renders a flush-left, padding-free help view for any cobra
// command. Use this as the fallback in custom HelpFunc implementations
// instead of cobra's default (which pads 2 spaces on every line and inserts
// blank lines around every section).
//
// Sections rendered: description, usage, aliases, examples, subcommands,
// local flags, global flags. Sections are omitted when empty.
func MinimalHelp(cmd *cobra.Command, args []string) {
	w := cmd.OutOrStdout()
	fmt.Fprintln(w)

	// Description — prefer Long, fall back to Short.
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
			fmt.Fprintln(w, Faint(cmd.UseLine()))
		}
		if cmd.HasAvailableSubCommands() {
			fmt.Fprintln(w, Faint(cmd.CommandPath()+" [command]"))
		}
		fmt.Fprintln(w)
	}

	// Aliases.
	if len(cmd.Aliases) > 0 {
		fmt.Fprintf(w, "%s\n", Strong("Aliases:"))
		fmt.Fprintln(w, cmd.NameAndAliases())
		fmt.Fprintln(w)
	}

	// Examples.
	if cmd.HasExample() {
		fmt.Fprintf(w, "%s\n", Strong("Examples:"))
		// Cobra examples are often indented 2 spaces; strip to flush-left.
		for _, line := range strings.Split(strings.TrimRight(cmd.Example, "\n"), "\n") {
			fmt.Fprintln(w, strings.TrimPrefix(line, "  "))
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
			fmt.Fprintf(w, "%s %s\n", PadVisible(colorizeCmd(c.Name()), HelpColWidth), Faint(c.Short))
		}
		fmt.Fprintln(w)
	}

	// Local flags — FlagUsages comes pre-indented, strip leading spaces.
	if cmd.HasAvailableLocalFlags() {
		fmt.Fprintf(w, "%s\n", Strong("Flags:"))
		writeStrippedFlagUsages(w, cmd.LocalFlags().FlagUsages())
		fmt.Fprintln(w)
	}

	// Global flags.
	if cmd.HasAvailableInheritedFlags() {
		fmt.Fprintf(w, "%s\n", Strong("Global flags:"))
		writeStrippedFlagUsages(w, cmd.InheritedFlags().FlagUsages())
		fmt.Fprintln(w)
	}
}

// writeStrippedFlagUsages writes FlagUsages output with its leading
// whitespace removed. Pflag pads every flag line with 2 spaces; we drop
// that so the output is flush-left like the rest of our help.
func writeStrippedFlagUsages(w io.Writer, usages string) {
	usages = strings.TrimRight(usages, "\n")
	if usages == "" {
		return
	}
	for _, line := range strings.Split(usages, "\n") {
		fmt.Fprintln(w, strings.TrimPrefix(line, "  "))
	}
}
