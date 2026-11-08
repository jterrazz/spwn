package catalog_test

import (
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
)

func TestAMacIsShownByItsModelsName(t *testing.T) {
	for _, c := range []struct{ id, want string }{
		{"Mac16,10", "Mac mini"},
		{"Mac16,8", "MacBook Pro"},
		{"Mac15,13", "MacBook Air"},
		{"MacBookPro18,3", "MacBook Pro"},
		{"MacBookAir10,1", "MacBook Air"},
		{"Macmini9,1", "Mac mini"},
		{"iMac21,1", "iMac"},
		{"Mac99,1", "Mac"},
		{"", "Mac"},
	} {
		if got := catalog.MacModelName(c.id); got != c.want {
			t.Errorf("MacModelName(%q) = %q, want %q", c.id, got, c.want)
		}
	}
}

func TestEveryModelIdentifierHasOneName(t *testing.T) {
	seen := map[string]bool{}
	for _, m := range catalog.MacModels {
		key := m.ID + "|" + m.Prefix
		if (m.ID == "") == (m.Prefix == "") || m.Name == "" || seen[key] {
			t.Errorf("%+v: a row names one identifier or one prefix, once, and its name", m)
		}
		seen[key] = true
	}
}
