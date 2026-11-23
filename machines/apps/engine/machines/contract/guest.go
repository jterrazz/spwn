package contract

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/netip"
)

var (
	// value: a dial its Mac refused because it has not approved station for
	// Local Network; the dial's own error is wrapped beside it.
	ErrLocalNetwork = errors.New("the Mac has not approved Machines for Local Network")
	// value: a guest that answered and took none of the keys or the password
	// it was offered.
	ErrLoginRefused = errors.New("the guest refused the login")
)

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
	// value: where the command's output streams; nil keeps it in the result.
	Stdout, Stderr io.Writer
}

type GuestResult struct {
	Code           int
	Stdout, Stderr string
}

type Guest interface {
	Exec(ctx context.Context, c GuestCommand) (GuestResult, error)
	Forward(ctx context.Context, addr string, port int) (io.ReadWriteCloser, error)
	Put(ctx context.Context, path string, data []byte, mode fs.FileMode) error
	Close() error
	Done() <-chan struct{}
}
