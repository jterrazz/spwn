package engine

import (
	"strconv"
	"strings"
	"testing"
)

func TestTheLogKeepsItsLastFiveHundredLinesAndDropsABlankOne(t *testing.T) {
	var b strings.Builder
	for i := range 700 {
		b.WriteString("line " + strconv.Itoa(i) + "\n\n")
	}
	got := LogOf([]byte(b.String()), false)
	if len(got.Lines) != LogLines || got.Lines[0] != "line 200" || got.Lines[LogLines-1] != "line 699" || got.Empty != "" {
		t.Errorf("log = %d lines from %q to %q, empty %q; want 500 from line 200 to line 699", len(got.Lines), got.Lines[0], got.Lines[len(got.Lines)-1], got.Empty)
	}
}

func TestATailCutInsideALineDropsThatLine(t *testing.T) {
	got := LogOf([]byte("rt of a line\nwhole line\n"), true)
	if len(got.Lines) != 1 || got.Lines[0] != "whole line" {
		t.Errorf("lines = %q; want the whole line alone", got.Lines)
	}
	if kept := LogOf([]byte("first line\n"), false); len(kept.Lines) != 1 {
		t.Errorf("an uncut tail lost its first line: %q", kept.Lines)
	}
}

func TestALogWithNoLineSaysSoInItsOwnWords(t *testing.T) {
	for _, tail := range []string{"", "\n \r\n"} {
		if got := LogOf([]byte(tail), false); got.Empty != LogEmpty || got.Lines == nil || len(got.Lines) != 0 {
			t.Errorf("log of %q = %+v; want no lines and %q", tail, got, LogEmpty)
		}
	}
}
