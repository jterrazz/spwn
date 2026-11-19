package contract

import (
	"context"
	"errors"
)

var ErrUnreachable = errors.New("the Mac does not answer")

type RestartRefusedError struct{ Detail string }

func (e *RestartRefusedError) Error() string { return "the Mac refused to restart: " + e.Detail }

type Unlocker interface {
	// invariant: the owner types the password into macOS's own pre-boot prompt
	// on the terminal, never into station; a Mac that does not answer is
	// ErrUnreachable and no session opens. Tailscale starts only once the
	// disk is unlocked, so the Mac is dialed at lan, its last address on your
	// LAN; an empty lan dials its name.
	Unlock(ctx context.Context, mac, lan string) error
	// invariant: a planned restart unlocks itself once; the Mac dropping the
	// connection to reboot is its success.
	PlannedRestart(ctx context.Context, mac string) error
}
