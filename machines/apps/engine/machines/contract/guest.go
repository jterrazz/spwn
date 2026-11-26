package contract

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/netip"
	"strconv"
)

var (
	// value: a guest that answered and took none of the keys or the password
	// it was offered.
	ErrLoginRefused = errors.New("the guest refused the login")
	// value: a guest that answered with a host key other than the one its
	// VM's known hosts pin.
	ErrHostKeyChanged = errors.New("the guest answered with another host key")
)

type NoRouteError struct {
	// value: a dial a Mac refused with the errno macOS answers a process
	// it keeps off the local network, as the dial said it. It says nothing
	// of why: only a read of the Mac's Local Network approval does.
	Err error
}

func (e *NoRouteError) Error() string { return e.Err.Error() }

func (e *NoRouteError) Unwrap() error { return e.Err }

type LockdownError struct {
	// value: a dial its VM's lockdown failed at: why, and whether no start
	// can run the lockdown again: the guest now takes neither station's key
	// nor its image's login, or answers with another host key.
	Cause string
	Gone  bool
}

func (e *LockdownError) Error() string { return e.Cause }

// value: the most of a command's output, its stdout and its stderr each,
// an Exec keeps in its result; what streams to a writer it is given has
// no bound.

const GuestOutputMost = 8 << 20

type OutputTooLargeError struct {
	// value: the VM whose command printed past GuestOutputMost with no
	// writer to take it: the command is ended, its result holding its
	// output up to the most.
	VM string
}

func (e *OutputTooLargeError) Error() string {
	return e.VM + "'s command printed more than " + strconv.Itoa(GuestOutputMost>>20) + " MiB, the most a command's output is kept, so it was ended"
}

type GuestDialer interface {
	Dial(ctx context.Context, t GuestTarget) (Guest, error)
}

type GuestTarget struct {
	VM      string
	Address netip.AddrPort
	User    string
	// value: the private key file the guest trusts; empty when Password is set.
	Key string
	// value: a base image's own account password, before its key is installed.
	Password string
	// value: the VM's own known-hosts file, its host key pinned there on first
	// sight; empty only for a build VM, a fresh clone whose image key is the trust.
	KnownHosts string
}

type GuestCommand struct {
	Line string
	// invariant: a secret reaches the guest by Stdin, never inside Line.
	Stdin io.Reader
	// value: where the command's output streams; nil keeps it in the
	// result, up to GuestOutputMost.
	Stdout, Stderr io.Writer
}

type GuestResult struct {
	Code           int
	Stdout, Stderr string
}

type Guest interface {
	// invariant: an Exec that errors, its connection lost or its ctx ended,
	// answers the output its command wrote before.
	Exec(ctx context.Context, c GuestCommand) (GuestResult, error)
	Forward(ctx context.Context, addr string, port int) (io.ReadWriteCloser, error)
	Put(ctx context.Context, path string, data []byte, mode fs.FileMode) error
	Close() error
	Done() <-chan struct{}
}
