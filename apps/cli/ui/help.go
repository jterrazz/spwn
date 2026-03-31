package ui

import (
	"fmt"
	"io"
	"strings"
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

// RenderGroupedHelp writes grouped command help to w.
func RenderGroupedHelp(w io.Writer, header string, groups []HelpGroup, usage string, flags string) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s\n", header)
	fmt.Fprintln(w)

	if usage != "" {
		fmt.Fprintf(w, "  %s\n", Strong("Usage:"))
		fmt.Fprintf(w, "    %s\n", Faint(usage))
		fmt.Fprintln(w)
	}

	for _, g := range groups {
		fmt.Fprintf(w, "  %s\n", Strong(g.Title+":"))
		for _, c := range g.Commands {
			fmt.Fprintf(w, "    %s %s\n", PadVisible(colorizeCmd(c.Name), 28), Faint(c.Desc))
		}
		fmt.Fprintln(w)
	}

	if flags != "" {
		fmt.Fprintf(w, "  %s\n", Faint(flags))
		fmt.Fprintln(w)
	}
}
