package main

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestAPathIsMarkedWithTheMachinesItLivesOn(t *testing.T) {
	for _, c := range []struct {
		kinds []catalog.MachineKind
		want  string
	}{
		{nil, ""},
		{[]catalog.MachineKind{catalog.Mac, catalog.Linux, catalog.MacOS}, ""},
		{[]catalog.MachineKind{catalog.Mac, catalog.MacOS}, "macOS"},
		{[]catalog.MachineKind{catalog.Linux, catalog.MacOS}, "vm"},
		{[]catalog.MachineKind{catalog.Linux}, "linux"},
		{[]catalog.MachineKind{catalog.Mac}, "Mac"},
	} {
		if got := on(c.kinds); got != c.want {
			t.Errorf("on(%v) = %q, want %q", c.kinds, got, c.want)
		}
	}
}
