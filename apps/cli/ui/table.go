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
}

// NewTable creates a table with the given headers.
func NewTable(headers ...string) *Table {
	return &Table{
		w:       os.Stderr,
		headers: headers,
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

// PadVisible pads s so that its visible width (ignoring ANSI codes) reaches w.
func PadVisible(s string, w int) string {
	return padVisible(s, w)
}

func padVisible(s string, w int) string {
	vis := visibleWidth(s)
	if vis >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vis)
}

// statusDisplay turns a bare status word into a colored,
// icon-prefixed display string for a STATUS column cell. Covers
// both the world-lifecycle vocabulary (running/idle/unattached)
// and the auth-lifecycle vocabulary (connected/disconnected/
// not configured/error). Cells whose first rune is an ANSI escape
// are assumed to be pre-formatted by the caller and pass through
// untouched - that's the opt-out for custom status strings (e.g.
// inline error messages from `auth check`).
func statusDisplay(cell string) string {
	if strings.HasPrefix(cell, "\x1b[") {
		return cell
	}
	switch cell {
	case "running", "connected":
		return green.Sprint("● " + cell)
	case "idle":
		return yellow.Sprint("◌ " + cell)
	case "not configured", "disconnected", "unattached":
		return faint("○ " + cell)
	default:
		return faint("○ " + cell)
	}
}

// Render prints the table with dimmed headers and 2-space indent.
func (t *Table) Render() {
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
			w := visibleWidth(cell)
			// Account for the status icon prefix the renderer adds
			// for unformatted cells. Pre-formatted cells (those
			// starting with an ANSI escape) are passed through so
			// their visible width is already accurate.
			if t.headers[i] == "STATUS" && !strings.HasPrefix(cell, "\x1b[") {
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
			display := cell
			if t.headers[i] == "STATUS" {
				display = statusDisplay(cell)
			}
			// Pad based on visible width to handle ANSI codes and status icons.
			fmt.Fprint(t.w, padVisible(display, widths[i]+gap))
		}
		fmt.Fprintln(t.w)
	}
	fmt.Fprintln(t.w)
}
