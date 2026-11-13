package contract

import "context"

type Password interface {
	// invariant: whether a line run as root asks the owner a password on
	// this machine, read without ever asking it and within a bound; a read
	// that cannot tell answers an error, and the plan then takes it as
	// asking one.
	Asked(ctx context.Context) (bool, error)
}
