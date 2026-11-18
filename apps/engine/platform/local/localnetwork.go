package local

import (
	"errors"
	"syscall"
)

var (
	// value: the errno of a route the system refuses, EHOSTUNREACH.
	ErrUnreachableRoute error = syscall.EHOSTUNREACH
)

func BlockedByLocalNetwork(goos string, err error) bool {
	// invariant: macOS refuses a connection into its local network, its VMs'
	// addresses included, with EHOSTUNREACH while the Mac has not approved the
	// process for Local Network; the same errno on any other system is a route
	// that is down, never an approval.
	return goos == "darwin" && errors.Is(err, ErrUnreachableRoute)
}
