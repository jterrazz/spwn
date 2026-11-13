package contract

import (
	"context"
	"io"
	"io/fs"
	"net/netip"
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
