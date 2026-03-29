package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
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

// pad returns s padded with trailing spaces to width w.
// This avoids %-*s issues with ANSI escape codes.
func pad(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

// Render prints the table with dimmed headers and 2-space indent.
func (t *Table) Render() {
	if t.mode == ModeQuiet || t.mode == ModeJSON {
		return
	}

	// Calculate column widths from plain text (headers + all rows).
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = len(h)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
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
					display = green.Sprint(cell)
				case "idle":
					display = faint(cell)
				}
			}
			// Pad based on plain-text cell length, then replace plain text with styled version.
			fmt.Fprint(t.w, pad(display, widths[i]+gap+(len(display)-len(cell))))
		}
		fmt.Fprintln(t.w)
	}
	fmt.Fprintln(t.w)
}
