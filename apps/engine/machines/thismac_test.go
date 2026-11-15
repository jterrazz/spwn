package machines_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines"
)

func TestThisMacName(t *testing.T) {
	for _, c := range []struct{ preference, hostname, want string }{
		{"studio-laptop", "Desk-One", "studio-laptop"},
		{"", "Desk-One", "desk-one"},
		{"", "", ""},
	} {
		if got := machines.ThisMacName(c.preference, c.hostname); got != c.want {
			t.Errorf("ThisMacName(%q, %q) = %q, want %q", c.preference, c.hostname, got, c.want)
		}
	}
}
