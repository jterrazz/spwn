package catalog_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestTheSSHCommandIsOneShellWordWhateverItsUserAndHost(t *testing.T) {
	for _, c := range []struct{ user, host, want string }{
		{"jterrazz.ops", "api", "ssh jterrazz.ops@api"},
		{"", "linux-3", "ssh linux-3"},
		{"you;touch pwned", "api", "ssh 'you;touch pwned'@api"},
		{"you\nProxyCommand x", "api", "ssh 'you\nProxyCommand x'@api"},
		{"it's", "a b", `ssh 'it'\''s'@'a b'`},
		{"", "$(touch pwned)", "ssh '$(touch pwned)'"},
	} {
		if got := catalog.SSHCommand(c.user, c.host); got != c.want {
			t.Errorf("SSHCommand(%q, %q) = %q; want %q", c.user, c.host, got, c.want)
		}
	}
}
