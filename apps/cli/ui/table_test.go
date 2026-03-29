package ui

import (
	"bytes"
	"strings"
	"testing"
)

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
