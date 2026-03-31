package ui

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

// stripANSI removes ANSI escape sequences for test assertions.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func TestTable_BasicRender(t *testing.T) {
	tbl := NewTable(ModeNormal, "NAME", "VALUE")
	tbl.w = &bytes.Buffer{}
	tbl.AddRow("foo", "bar")
	tbl.AddRow("hello", "world")
	tbl.Render()

	out := tbl.w.(*bytes.Buffer).String()
	if !strings.Contains(out, "NAME") {
		t.Error("output should contain header NAME")
	}
	if !strings.Contains(out, "VALUE") {
		t.Error("output should contain header VALUE")
	}
	if !strings.Contains(out, "foo") {
		t.Error("output should contain row value foo")
	}
	if !strings.Contains(out, "world") {
		t.Error("output should contain row value world")
	}
}

func TestTable_ColumnAlignment(t *testing.T) {
	tbl := NewTable(ModeNormal, "ID", "CONFIG", "STATUS")
	buf := &bytes.Buffer{}
	tbl.w = buf
	tbl.AddRow("w-default-13182", "default", "running")
	tbl.AddRow("w-prod-99", "production", "idle")
	tbl.Render()

	out := stripANSI(buf.String())
	// Output starts with \n, so trim leading newline then split
	out = strings.TrimPrefix(out, "\n")
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 rows), got %d: %v", len(lines), lines)
	}

	// All lines should start with 2-space indent
	for i, line := range lines {
		if !strings.HasPrefix(line, "  ") {
			t.Errorf("line %d should start with 2-space indent: %q", i, line)
		}
	}

	// The STATUS column should start at the same position across rows.
	// Status values now have icon prefixes (● / ◌), so find the icon position
	// which marks the start of the STATUS column.
	header := lines[0]
	row1 := lines[1]
	row2 := lines[2]

	hIdx := strings.Index(header, "STATUS")
	// Find the status icon which marks the column start
	idx1 := strings.Index(row1, "●")
	idx2 := strings.Index(row2, "◌")
	if hIdx < 0 || idx1 < 0 || idx2 < 0 {
		t.Fatalf("could not find expected column values in:\n%s\n%s\n%s", header, row1, row2)
	}
	// All should start at the same column offset
	if idx1 != hIdx || idx2 != hIdx {
		t.Errorf("STATUS column not aligned: header at %d, row1 icon at %d, row2 icon at %d", hIdx, idx1, idx2)
	}
}

func TestTable_ColumnWidthExpands(t *testing.T) {
	tbl := NewTable(ModeNormal, "ID", "STATUS")
	tbl.w = &bytes.Buffer{}
	tbl.AddRow("a-very-long-identifier", "running")
	tbl.Render()

	out := tbl.w.(*bytes.Buffer).String()
	if !strings.Contains(out, "a-very-long-identifier") {
		t.Error("long value should not be truncated")
	}
}

func TestTable_EmptyTable(t *testing.T) {
	tbl := NewTable(ModeNormal, "A", "B")
	tbl.w = &bytes.Buffer{}
	tbl.Render()

	out := tbl.w.(*bytes.Buffer).String()
	// Should still have headers
	if !strings.Contains(out, "A") {
		t.Error("empty table should still render headers")
	}
	// Should not have data rows — just header + surrounding newlines
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) > 1 {
		t.Errorf("empty table should have only 1 content line (header), got %d", len(lines))
	}
}

func TestTable_QuietModeSuppressesOutput(t *testing.T) {
	tbl := NewTable(ModeQuiet, "NAME")
	tbl.w = &bytes.Buffer{}
	tbl.AddRow("test")
	tbl.Render()

	out := tbl.w.(*bytes.Buffer).String()
	if out != "" {
		t.Errorf("quiet mode should produce no output, got %q", out)
	}
}

func TestTable_JSONModeSuppressesOutput(t *testing.T) {
	tbl := NewTable(ModeJSON, "NAME")
	tbl.w = &bytes.Buffer{}
	tbl.AddRow("test")
	tbl.Render()

	out := tbl.w.(*bytes.Buffer).String()
	if out != "" {
		t.Errorf("JSON mode should produce no output, got %q", out)
	}
}

func TestTable_MultipleRows(t *testing.T) {
	tbl := NewTable(ModeNormal, "COL1", "COL2", "COL3")
	tbl.w = &bytes.Buffer{}
	tbl.AddRow("a", "b", "c")
	tbl.AddRow("d", "e", "f")
	tbl.AddRow("g", "h", "i")
	tbl.Render()

	out := tbl.w.(*bytes.Buffer).String()
	for _, val := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"} {
		if !strings.Contains(out, val) {
			t.Errorf("output missing expected value %q", val)
		}
	}
}

func TestTable_ConsistentColumnWidth(t *testing.T) {
	tbl := NewTable(ModeNormal, "NAME", "VALUE")
	buf := &bytes.Buffer{}
	tbl.w = buf
	tbl.AddRow("short", "1")
	tbl.AddRow("a-much-longer-name", "2")
	tbl.AddRow("mid", "3")
	tbl.Render()

	out := stripANSI(buf.String())
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}

	// VALUE column values ("1", "2", "3") should all start at the same position.
	// Find the position of the value digit in each data row.
	positions := make([]int, 0, 3)
	for _, line := range lines[1:] {
		// Find the value column: after the NAME column gap
		// The value is at a fixed offset from the start
		for i := len(line) - 1; i >= 0; i-- {
			if line[i] >= '1' && line[i] <= '3' {
				positions = append(positions, i)
				break
			}
		}
	}
	if len(positions) != 3 {
		t.Fatalf("expected 3 value positions, got %d", len(positions))
	}
	for i := 1; i < len(positions); i++ {
		if positions[i] != positions[0] {
			t.Errorf("VALUE column not consistently aligned: positions %v", positions)
			break
		}
	}
}
