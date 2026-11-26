package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/approvals"
)

type LocalNetwork interface {
	// value: one read of macOS's Local Network approval of this process,
	// now: approved, refused, or unknown where it cannot tell; never
	// checking, which is the family's judgement of a refusal's age.
	Read(ctx context.Context) approvals.State
}
