package contract

import (
	"context"
	"errors"
)

var ErrBadHardwareAddress = errors.New("not a hardware address")

type Waker interface {
	// invariant: SPEC B5.14: one Wake-on-LAN magic packet naming ether,
	// broadcast from this Mac on its LAN; an address that does not parse is
	// ErrBadHardwareAddress and nothing is sent. A packet sent says nothing
	// of whether the Mac woke: its answer does.
	Wake(ctx context.Context, ether string) error
}
