package accounts_test

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
)

func TestExpired(t *testing.T) {
	now := time.Date(2026, 11, 4, 14, 0, 0, 0, time.UTC)
	ends := func(name string, d time.Duration) accounts.Share {
		return accounts.Share{Name: name, At: now.Add(-time.Hour), Until: now.Add(d)}
	}
	held := []accounts.Share{ends("session:github", -time.Minute), ends("operator:claude:oauth-token", 0), ends("session:openpanel", time.Second)}
	got := accounts.Expired(held, now)
	if len(got) != 2 || got[0].Name != "session:github" || got[1].Name != "operator:claude:oauth-token" {
		t.Fatalf("Expired = %+v; want the share past its until and the one ending now, never the one a second left", got)
	}
}
