package machine_test

import (
	"encoding/json"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func TestAKindIsReadAndWrittenAsItsOwnWordAlone(t *testing.T) {
	// invariant: no older word is translated: "mac" stays what it says, a
	// word that is no kind.
	for _, kind := range []machine.Kind{machine.Computer, machine.Server, machine.Workspace, "mac"} {
		var k machine.Kind
		out, _ := json.Marshal(kind)
		if err := json.Unmarshal(out, &k); err != nil || k != kind {
			t.Fatalf("%q read as %q, %v", kind, k, err)
		}
	}
}

func TestAComputerIsCalledAMacOnlyWhenItsSystemIsMacOS(t *testing.T) {
	for _, c := range []struct {
		kind machine.Kind
		os   machine.OS
		want string
	}{
		{machine.Computer, machine.MacOS, "Mac"},
		{machine.Computer, "", "Mac"},
		{machine.Computer, machine.Linux, "computer"},
		{machine.Workspace, machine.MacOS, "workspace"},
		{machine.Server, machine.Linux, "server"},
	} {
		if got := c.kind.Word(c.os); got != c.want {
			t.Fatalf("a %s on %q is called %q, want %q", c.kind, c.os, got, c.want)
		}
	}
}
