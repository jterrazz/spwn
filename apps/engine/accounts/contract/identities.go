package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type Identities interface {
	// value: whether this machine has a row for the provider at all: a
	// provider read through its configuration offers one only once that
	// configuration names where its logins are kept.
	Offers(p catalog.Provider) bool
	// invariant: a read that does not answer within the provider's budget is
	// accounts.Late; a tool that is not there is absent and missing.
	Read(ctx context.Context, p catalog.Provider) (accounts.Identity, error)
}
