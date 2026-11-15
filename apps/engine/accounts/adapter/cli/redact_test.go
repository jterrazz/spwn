package cli_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
)

func TestRedactDropsEveryTokenShape(t *testing.T) {
	shapes := []string{
		"ghp_abcdefghij0123456789", "gho_x", "ghs_abc", "ghu_abc", "ghr_abc", "github_pat_11ABCDEF_xyz",
		"sk-ant-api03-abcdef", "npm_abcdefghij0123456789abcdefghij012345", "xoxb-1234-5678", "xoxp-1-2",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.c2lnbmF0dXJl",
		"0123456789abcdef0123456789abcdef", "QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo0MTIz",
	}
	for _, s := range shapes {
		if kept, redacted := accounts.Redact(s); kept != accounts.Unknown || !redacted {
			t.Errorf("%q survived as %q", s, kept)
		}
	}
	for _, s := range []string{"you@example.com", "Jean-Baptiste Terrazzoni", "SHA256:76MCcfWNZv0Ixcc4fM0b6Ue1o8l7W1Xo0n3zKhK1Vy4", "registry.example.com", "read:org"} {
		if kept, redacted := accounts.Redact(s); kept != s || redacted {
			t.Errorf("an identity %q was taken for a token", s)
		}
	}
}
