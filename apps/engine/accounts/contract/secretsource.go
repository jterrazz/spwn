package contract

import "context"

type SecretSource interface {
	// value: the secrets another tool keeps, exported at once as one JSON
	// object of name to value, the form that tool imports them back from.
	// invariant: a name it does not keep fails the whole export.
	Export(ctx context.Context, names []string) ([]byte, error)
}
