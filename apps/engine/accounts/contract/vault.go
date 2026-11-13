package contract

import "context"

type Vault interface {
	// invariant: a name the vault does not keep answers accounts.ErrNotKept.
	Get(ctx context.Context, name string) ([]byte, error)
	// invariant: the secret is never in a command line nor a file station
	// writes; putting a name again replaces its secret.
	Put(ctx context.Context, name string, secret []byte) error
	// invariant: forgetting a name the vault does not keep answers
	// accounts.ErrNotKept, and leaves the vault as it was.
	Forget(ctx context.Context, name string) error
	// invariant: every name kept, sorted.
	Names(ctx context.Context) ([]string, error)
}
