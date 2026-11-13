package cli_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
)

func TestRedactDropsEveryTokenShape(t *testing.T) {
	shapes := []string{
		"ghp_abcdefghij0123456789", "gho_x", "ghs_abc", "ghu_abc", "ghr_abc", "github_pat_11ABCDEF_xyz",
		"sk-ant-api03-abcdef", "npm_abcdefghij0123456789", "xoxb-1234-5678", "xoxp-1-2",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.c2lnbmF0dXJl",
		"0123456789abcdef0123456789abcdef", "QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo0MTIz",
	}
	for _, s := range shapes {
		a := accounts.RedactAccount(accounts.Account{Identity: []string{s}, Scopes: []string{"repo", s}, Detail: s, Notes: []string{s}})
		if a.Identity[0] != accounts.Unknown || a.Scopes[1] != accounts.Unknown || a.Detail != accounts.Unknown || a.Notes[0] != accounts.Unknown || !a.Redacted {
			t.Errorf("%q survived: %+v", s, a)
		}
		if a.Scopes[0] != "repo" {
			t.Errorf("a plain scope beside %q was dropped", s)
		}
	}
	for _, s := range []string{"admin@jterrazz.com", "Jean-Baptiste Terrazzoni", "SHA256:76MCcfWNZv0Ixcc4fM0b6Ue1o8l7W1Xo0n3zKhK1Vy4", "registry.jterrazz.com", "read:org"} {
		if a := accounts.RedactAccount(accounts.Account{Identity: []string{s}}); a.Identity[0] != s || a.Redacted {
			t.Errorf("an identity %q was taken for a token", s)
		}
	}
}
