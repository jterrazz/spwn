package catalog

import (
	"strings"
	"testing"
)

func TestAMountReachesNoFolderItsSideKeepsOut(t *testing.T) {
	// invariant: a mount folder is refused when it climbs, is a whole home
	// or the folder of the homes, lies in a system folder of the VM, or
	// covers or enters a folder its side keeps out; every other folder of
	// a home or of the VM's own tree is a mount's.
	for _, c := range []struct {
		computer, guest, refused string
	}{
		{"~/code", "~/code", ""},
		{"/Volumes/data", "/srv/data", ""},
		{"~/data", "/opt/data", ""},
		{"~/data", "/mnt/data", ""},
		{"~/data", "/data", ""},
		{"~/.config/zed", "/home/admin/.config/zed", ""},
		{"/Users/me/code", "~/.local/share/notes", ""},
		{"~/notes/../.ssh", "~/notes", "climbs with .."},
		{"~/notes", "/srv/../etc", "climbs with .."},
		{"~", "~/home", "is a whole home"},
		{"/Users/me", "~/home", "is a whole home"},
		{"~/notes", "~", "is a whole home"},
		{"~/notes", "/home/admin", "is a whole home"},
		{"/", "~/root", "holds every home"},
		{"/Users", "~/users", "holds every home"},
		{"~/notes", "/home", "holds every home"},
		{"~/notes", "/", "holds every home"},
		{"~/notes", "/etc", "system folder /etc"},
		{"~/notes", "/usr/local/notes", "system folder /usr"},
		{"~/notes", "/tmp", "system folder /tmp"},
		{"~/notes", "/tmp/notes", "system folder /tmp"},
		{"~/notes", "/run/notes", "system folder /run"},
		{"~/notes", "/root/notes", "system folder /root"},
		{"~/notes", "/var/lib/notes", "system folder /var"},
		{"~/Library", "~/lib", "is in ~/Library"},
		{"/Users/me/Library/Keychains", "~/keys", "is in ~/Library"},
		{"~/.ssh", "~/ssh", "is in ~/.ssh"},
		{"~/.config", "~/config", "holds ~/.config/station"},
		{"~/.local", "~/local", "holds ~/.local/state/station"},
		{"~/.local/state/station/logs", "~/logs", "is in ~/.local/state/station"},
		{"~/notes", "~/.ssh", "is in ~/.ssh"},
		{"~/notes", "/home/admin/.ssh/keys", "is in ~/.ssh"},
		{"~/notes", "~/.config", "holds ~/.config/station"},
		{"~/notes", "~/.config/station/machinefiles", "is in ~/.config/station"},
		{"~/notes", "~/.local/state", "holds ~/.local/state/station"},
	} {
		got := checkMounts([]Mount{{Computer: c.computer, Guest: c.guest}})
		if c.refused == "" && got != "" || c.refused != "" && !strings.Contains(got, c.refused) {
			t.Errorf("%s at %s: %q; want %q", c.computer, c.guest, got, c.refused)
		}
	}
}

func TestTwoMountsAtOneGuestFolderAreReadWithTheHomeOpened(t *testing.T) {
	for _, guests := range [][2]string{{"~/notes", "~/notes/"}, {"~/notes", "/home/admin/notes"}, {"/data", "/data/."}} {
		got := checkMounts([]Mount{{Computer: "~/a", Guest: guests[0]}, {Computer: "~/b", Guest: guests[1]}})
		if !strings.Contains(got, "two mounts share the guest folder") {
			t.Errorf("%s and %s: %q; want them refused as one folder", guests[0], guests[1], got)
		}
	}
}
