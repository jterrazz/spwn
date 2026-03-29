package ui

import (
	"fmt"
	"io"
	"os"
)

// Table formats columnar data with auto-width and styled headers.
type Table struct {
	w       io.Writer
	headers []string
	widths  []int
	rows    [][]string
	mode    Mode
}

// NewTable creates a table with the given headers.
func NewTable(mode Mode, headers ...string) *Table {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	return &Table{
		w:       os.Stderr,
		headers: headers,
		widths:  widths,
		mode:    mode,
	}
}

// AddRow adds a row to the table. The number of columns must match the headers.
func (t *Table) AddRow(cols ...string) {
	for i, c := range cols {
		if i < len(t.widths) && len(c) > t.widths[i] {
			t.widths[i] = len(c)
		}
	}
	t.rows = append(t.rows, cols)
}

// Render prints the table with bold headers and 2-space indent.
func (t *Table) Render() {
	if t.mode == ModeQuiet || t.mode == ModeJSON {
		return
	}

	// Print header
	fmt.Fprintf(t.w, "\n  ")
	for i, h := range t.headers {
		fmt.Fprintf(t.w, "%-*s ", t.widths[i]+2, strong(h))
	}
	fmt.Fprintln(t.w)

	// Print rows
	for _, row := range t.rows {
		fmt.Fprintf(t.w, "  ")
		for i, col := range row {
			if i < len(t.widths) {
				// Color status column
				if t.headers[i] == "STATUS" {
					switch col {
					case "running":
						col = green.Sprint(col)
					case "idle":
						col = faint(col)
					}
				}
				fmt.Fprintf(t.w, "%-*s ", t.widths[i]+2, col)
			}
		}
		fmt.Fprintln(t.w)
	}
	fmt.Fprintln(t.w)
}
