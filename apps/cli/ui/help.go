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

// HelpColWidth is the column width used to pad command names in grouped help.
// Entries wider than this still render — they just push past the padding.
const HelpColWidth = 32

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
			if c.Desc == "" {
				fmt.Fprintf(w, "    %s\n", colorizeCmd(c.Name))
			} else {
				fmt.Fprintf(w, "    %s %s\n", PadVisible(colorizeCmd(c.Name), HelpColWidth), Faint(c.Desc))
			}
		}
		fmt.Fprintln(w)
	}

	if flags != "" {
		fmt.Fprintf(w, "  %s\n", Faint(flags))
		fmt.Fprintln(w)
	}
}
