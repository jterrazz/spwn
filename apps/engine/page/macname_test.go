package page_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/page"
)

func TestAMacIsNamedWithoutTheUserBothNamesStartWith(t *testing.T) {
	for _, c := range []struct{ name, self, want string }{
		{"jterrazz-macbook-pro", "jterrazz-macbook-pro", page.ThisMac},
		{"jterrazz-mac-mini", "jterrazz-macbook-pro", "mac-mini"},
		{"mac-mini", "jterrazz-macbook-pro", "mac-mini"},
		{"macbook-air", "macbook-pro", "macbook-air"},
		{"mac-mini", "macbook-pro", "mac-mini"},
		{"studio", "", "studio"},
	} {
		if got := page.MacName(c.name, c.self); got != c.want {
			t.Errorf("%s seen from %s is named %q, want %q", c.name, c.self, got, c.want)
		}
	}
}
