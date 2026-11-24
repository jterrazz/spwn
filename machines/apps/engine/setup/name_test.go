package setup_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/setup"
)

func TestAMacIsNamedByItsLocalHostnameNeverByWhatDHCPHandsIt(t *testing.T) {
	for _, c := range []struct{ local, hostname, want string }{
		{"Mac-Mini", "Mac-001.lan", "mac-mini"},
		{"Mac-Mini\n", "Mac-002.home", "mac-mini"},
		{"Jean's Desk 2", "desk.local", "jean-s-desk-2"},
		{"", "Mac-001.lan", "mac-001"},
		{"", "linux-1", "linux-1"},
		{"", "", ""},
	} {
		if got := setup.DefaultName(c.local, c.hostname); got != c.want {
			t.Errorf("DefaultName(%q, %q) = %q, want %q", c.local, c.hostname, got, c.want)
		}
	}
}
