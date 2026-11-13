package contract

import "context"

type Forge interface {
	// invariant: merging a PR open in folder's repository merges it; a PR
	// that is not open there is refused.
	Merge(ctx context.Context, folder string, number int) error
}
