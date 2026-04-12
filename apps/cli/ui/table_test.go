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
	tbl := NewTable("NAME", "VALUE")
	tbl.w = &bytes.Buffer{}
	tbl.AddRow("foo", "bar")
	tbl.AddRow("hello", "world")
	tbl.Render()

	out := tbl.w.(*bytes.Buffer).String()
	for _, want := range []string{"NAME", "VALUE", "foo", "world"} {
		if !strings.Contains(out, want) {
			t.Errorf("output should contain %q", want)
		}
	}
}

func TestTable_ColumnAlignment(t *testing.T) {
	tbl := NewTable("ID", "CONFIG", "STATUS")
	buf := &bytes.Buffer{}
	tbl.w = buf
	tbl.AddRow("w-default-13182", "default", "running")
	tbl.AddRow("w-prod-99", "production", "idle")
	tbl.Render()

	out := stripANSI(buf.String())
	out = strings.TrimPrefix(out, "\n")
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 rows), got %d: %v", len(lines), lines)
	}

	// All lines should start with a 2-space indent.
	for i, line := range lines {
		if !strings.HasPrefix(line, "  ") {
			t.Errorf("line %d should start with 2-space indent: %q", i, line)
		}
	}

	// STATUS column should align across header and data rows. The data rows
	// prefix their value with an icon (● / ◌), so use the icon index.
	header := lines[0]
	row1 := lines[1]
	row2 := lines[2]

	hIdx := strings.Index(header, "STATUS")
	idx1 := strings.Index(row1, "●")
	idx2 := strings.Index(row2, "◌")
	if hIdx < 0 || idx1 < 0 || idx2 < 0 {
		t.Fatalf("could not find expected column values in:\n%s\n%s\n%s", header, row1, row2)
	}
	if idx1 != hIdx || idx2 != hIdx {
		t.Errorf("STATUS column not aligned: header at %d, row1 icon at %d, row2 icon at %d", hIdx, idx1, idx2)
	}
}

func TestTable_ColumnWidthExpands(t *testing.T) {
	tbl := NewTable("ID", "STATUS")
	tbl.w = &bytes.Buffer{}
	tbl.AddRow("a-very-long-identifier", "running")
	tbl.Render()

	out := tbl.w.(*bytes.Buffer).String()
	if !strings.Contains(out, "a-very-long-identifier") {
		t.Error("long value should not be truncated")
	}
}

func TestTable_EmptyTable(t *testing.T) {
	tbl := NewTable("A", "B")
	tbl.w = &bytes.Buffer{}
	tbl.Render()

	out := tbl.w.(*bytes.Buffer).String()
	if !strings.Contains(out, "A") {
		t.Error("empty table should still render headers")
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) > 1 {
		t.Errorf("empty table should have only 1 content line (header), got %d", len(lines))
	}
}

func TestTable_MultipleRows(t *testing.T) {
	tbl := NewTable("COL1", "COL2", "COL3")
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
	tbl := NewTable("NAME", "VALUE")
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

	positions := make([]int, 0, 3)
	for _, line := range lines[1:] {
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
