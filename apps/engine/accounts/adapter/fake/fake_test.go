package fake_test

import (
	"context"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestVault(t *testing.T) {
	contracttest.Vault(t, func(*testing.T) contract.Vault {
		return fake.NewVault(&world.World{}, "mac-mini")
	})
}

func TestSecretSource(t *testing.T) {
	contracttest.SecretSource(t, func(*testing.T) contracttest.SecretSourceSubject {
		kept := map[string]string{"session:a": "value-a", "operator:claude:oauth-token": "value-token"}
		w := &world.World{Macs: map[string]*world.Mac{"mac-mini": {Secrets: kept}}}
		return contracttest.SecretSourceSubject{Source: fake.NewSecretSource(w, "mac-mini"), Kept: kept}
	})
}

func TestIdentities(t *testing.T) {
	contracttest.Identities(t, func(*testing.T) contracttest.IdentitiesSubject {
		w := &world.World{Macs: map[string]*world.Mac{"mac-mini": {Identities: map[string]accounts.Identity{
			catalog.ProviderGitHub: {State: catalog.StateOK, Who: []string{"jterrazz"}},
		}}}}
		return contracttest.IdentitiesSubject{
			Identities: fake.NewIdentities(w, "mac-mini"),
			SignedIn:   catalog.Provider{ID: catalog.ProviderGitHub}, Who: "jterrazz",
			Missing: catalog.Provider{ID: catalog.ProviderClaude},
		}
	})
}

func TestShareStore(t *testing.T) {
	contracttest.ShareStore(t, func(_ *testing.T, vms ...string) contract.ShareStore { return fake.NewShares(vms...) })
}

func TestSignIns(t *testing.T) {
	contracttest.SignIns(t, func(*testing.T) contracttest.SignInsSubject {
		in, out := accounts.Programs()[1], accounts.Programs()[0]
		w := &world.World{Macs: map[string]*world.Mac{"mac-mini": {Accounts: &world.Accounts{Programs: map[accounts.Program]accounts.Facts{
			in: {State: accounts.ProgramSignedIn, Who: "jterrazz", How: accounts.HowOwn},
		}}}}}
		at := time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)
		clock := fake.Clock{Now: func() time.Time { return at }, Sleep: func(ctx context.Context, _ time.Duration) error {
			<-ctx.Done()
			return ctx.Err()
		}}
		return contracttest.SignInsSubject{SignIns: fake.NewSignIns(w, "mac-mini", clock), SignedIn: in, SignedOut: out}
	})
}
