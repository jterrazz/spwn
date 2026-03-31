package ui

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Table formats columnar data with auto-width and styled headers.
type Table struct {
	w       io.Writer
	headers []string
	rows    [][]string
	mode    Mode
}

// NewTable creates a table with the given headers.
func NewTable(mode Mode, headers ...string) *Table {
	return &Table{
		w:       os.Stderr,
		headers: headers,
		mode:    mode,
	}
}

// AddRow adds a row to the table. The number of columns must match the headers.
func (t *Table) AddRow(cols ...string) {
	t.rows = append(t.rows, cols)
}

// ansiPattern matches ANSI escape sequences.
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// visibleWidth returns the number of visible runes in a string,
// ignoring ANSI escape sequences.
func visibleWidth(s string) int {
	return utf8.RuneCountInString(ansiPattern.ReplaceAllString(s, ""))
}

// pad returns s padded with trailing spaces to width w.
// This avoids %-*s issues with ANSI escape codes.
func pad(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

// padVisible pads s so that its visible width (ignoring ANSI codes) reaches w.
func padVisible(s string, w int) string {
	vis := visibleWidth(s)
	if vis >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vis)
}

// Render prints the table with dimmed headers and 2-space indent.
func (t *Table) Render() {
	if t.mode == ModeQuiet || t.mode == ModeJSON {
		return
	}

	// Calculate column widths from visible text (headers + all rows).
	// For STATUS columns, account for the added icon prefix.
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = len(h)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i >= len(widths) {
				continue
			}
			w := len(cell)
			// Account for status icon prefix that will be added during rendering.
			if t.headers[i] == "STATUS" {
				w += 2 // "● " / "◌ " / "○ " prefix
			}
			if w > widths[i] {
				widths[i] = w
			}
		}
	}

	const gap = 3 // spaces between columns

	// Print header (dimmed)
	fmt.Fprint(t.w, "\n  ")
	for i, h := range t.headers {
		fmt.Fprint(t.w, faint(pad(h, widths[i]+gap)))
	}
	fmt.Fprintln(t.w)

	// Print rows
	for _, row := range t.rows {
		fmt.Fprint(t.w, "  ")
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			// Colorize status values
			display := cell
			if t.headers[i] == "STATUS" {
				switch cell {
				case "running":
					display = green.Sprint("● " + cell)
				case "idle":
					display = yellow.Sprint("◌ " + cell)
				case "unattached":
					display = faint("○ " + cell)
				default:
					display = faint("○ " + cell)
				}
			}
			// Pad based on visible width to handle ANSI codes and status icons.
			fmt.Fprint(t.w, padVisible(display, widths[i]+gap))
		}
		fmt.Fprintln(t.w)
	}
	fmt.Fprintln(t.w)
}
