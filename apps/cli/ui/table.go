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

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func visibleWidth(s string) int {
	return utf8.RuneCountInString(ansiPattern.ReplaceAllString(s, ""))
}

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
			if t.headers[i] == "STATUS" && !strings.HasPrefix(cell, "\x1b[") {
				w += 2
			}
			if w > widths[i] {
				widths[i] = w
			}
		}
	}

	const gap = 3

	fmt.Fprint(t.w, "\n  ")
	for i, h := range t.headers {
		fmt.Fprint(t.w, faint(pad(h, widths[i]+gap)))
	}
	fmt.Fprintln(t.w)

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
			fmt.Fprint(t.w, padVisible(display, widths[i]+gap))
		}
		fmt.Fprintln(t.w)
	}
	fmt.Fprintln(t.w)
}
