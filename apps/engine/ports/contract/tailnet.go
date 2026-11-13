package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type Absent = ports.Absent // value: the error a Tailnet read answers on a machine without the app.

type Tailnet interface {
	// invariant: one read answers this node, its sharing and its peers, so a
	// page that shows all three asks the tailnet once.
	Self(ctx context.Context) (ports.Self, error)
	Ready(ctx context.Context) (ports.SharingState, error)
	// invariant: every share this node serves, private and public: what
	// an Audience opened, read back.
	Shared(ctx context.Context) ([]ports.Share, error)
	// invariant: never blocks a share: the certificate of the host the last
	// Ready named is asked for, and its issuance may take its time.
	Warm(ctx context.Context) error
}
