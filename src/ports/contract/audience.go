package contract

import (
	"context"

	"github.com/jterrazz/studio-station/src/ports"
)

type Audience interface {
	// invariant: a share is opened to the audience it names: private on the
	// tailnet (Tailscale Serve), public on the internet (Tailscale Funnel, on
	// the Funnel port the share names). A local share is never opened: the
	// Mac's loopback listener is the whole of it, so opening one changes
	// nothing and answers no URL. Every share is carried from this Mac's
	// loopback only, never from a wildcard or the tailnet address.
	Share(ctx context.Context, s ports.Share) (url string, err error)
	// invariant: closing a share removes it from the audience it was opened
	// to, and leaves every other share open.
	Unshare(ctx context.Context, s ports.Share) error
}
