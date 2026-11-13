package contracttest

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type IdentitiesSubject struct {
	Identities contract.Identities
	// value: a provider signed in as Who on this machine.
	SignedIn catalog.Provider
	Who      string
	// value: a provider whose tool is not on this machine.
	Missing catalog.Provider
}

func Identities(t *testing.T, subject func(t *testing.T) IdentitiesSubject) {
	ctx := context.Background()

	t.Run("a signed-in provider is offered and read by who it says is signed in", func(t *testing.T) {
		s := subject(t)
		if !s.Identities.Offers(s.SignedIn) {
			t.Fatalf("Offers(%s) = false", s.SignedIn.ID)
		}
		id, err := s.Identities.Read(ctx, s.SignedIn)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if id.Provider != s.SignedIn.ID || id.State != catalog.StateOK || len(id.Who) == 0 || id.Who[0] != s.Who {
			t.Fatalf("Read = %+v; want %s signed in as %s", id, s.SignedIn.ID, s.Who)
		}
	})

	t.Run("a provider whose program is not there reads absent, never an error", func(t *testing.T) {
		s := subject(t)
		id, err := s.Identities.Read(ctx, s.Missing)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if id.Provider != s.Missing.ID || id.State != catalog.StateAbsent || id.Reason != accounts.ReasonMissing {
			t.Fatalf("Read = %+v; want %s absent and missing", id, s.Missing.ID)
		}
	})
}
