package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/engine"
)

type Releases interface {
	// invariant: the latest release of station as GitHub gives it, drafts
	// and pre-releases never; no release answers the zero Tagged
	// with no error, and a GitHub that does not answer is an error.
	Latest(ctx context.Context) (engine.Tagged, error)
}

type ReleaseCache interface {
	// invariant: what the last Write left; none written reads as none
	// (false) with no error.
	Read(ctx context.Context) (engine.ReleaseKept, bool, error)
	Write(ctx context.Context, k engine.ReleaseKept) error
}
