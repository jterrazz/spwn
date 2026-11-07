package contract

import (
	"context"

	"github.com/jterrazz/studio-station/src/machines"
)

type CheckRunner interface {
	// invariant: Run is a server's "Check passes" step, run from its Mac:
	// nil once the URL answered 2xx or 3xx, or the host:port accepted a
	// connection, within machines.CheckBudget; any other end is its error.
	// Its answer is the status a URL answered (`200`), empty for a
	// connection, and how long the answer took.
	Run(ctx context.Context, t machines.CheckTarget) (machines.CheckAnswer, error)
}
