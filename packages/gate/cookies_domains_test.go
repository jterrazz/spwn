package gate

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// WriteDomainHints persists each provider's domain list to
// /credentials/<provider>/.domains so the gate-browser sidecar
// knows which hosts to attach cookies to. Without this hint a
// session opened with provider:"linkedin" would have nowhere to
// pin the li_at cookie.

func TestWriteDomainHints_WritesOneFilePerProvider(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	cs := NewCookieSync()
	cs.RegisterProvider(CookieProvider{
		Name:    "x",
		Domains: []string{"x.com", "twitter.com"},
		Cookies: []string{"auth_token", "ct0"},
	})
	cs.RegisterProvider(CookieProvider{
		Name:    "linkedin",
		Domains: []string{"linkedin.com"},
		Cookies: []string{"li_at"},
	})

	if err := cs.WriteDomainHints(); err != nil {
		t.Fatalf("WriteDomainHints: %v", err)
	}

	cases := map[string][]string{
		"x":        {"x.com", "twitter.com"},
		"linkedin": {"linkedin.com"},
	}
	for prov, want := range cases {
		raw, err := os.ReadFile(filepath.Join(tmp, "credentials", prov, ".domains"))
		if err != nil {
			t.Errorf("%s: %v", prov, err)
			continue
		}
		got := []string{}
		for _, l := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
			if l != "" {
				got = append(got, l)
			}
		}
		sort.Strings(got)
		sort.Strings(want)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s: domains = %v, want %v", prov, got, want)
		}
	}
}

func TestWriteDomainHints_EmptyRegistry_NoOp(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	cs := NewCookieSync()
	if err := cs.WriteDomainHints(); err != nil {
		t.Errorf("empty registry: %v", err)
	}
	if entries, err := os.ReadDir(filepath.Join(tmp, "credentials")); err == nil && len(entries) > 0 {
		t.Errorf("created %d entries for empty registry: %v", len(entries), entries)
	}
}

func TestWriteDomainHints_FilePermsRestrictive(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	cs := NewCookieSync()
	cs.RegisterProvider(CookieProvider{Name: "x", Domains: []string{"x.com"}, Cookies: []string{"a"}})
	if err := cs.WriteDomainHints(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(tmp, "credentials", "x", ".domains"))
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("file perms = %o, want 0600 (matches cookies.json convention)", mode)
	}
}
