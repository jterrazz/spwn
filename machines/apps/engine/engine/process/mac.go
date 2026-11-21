package process

import (
	"context"
	"time"
)

type Stream interface {
	// value: one stream the engine holds: a client of its socket, a VM port
	// it carries, a guest session.
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	Close() error
}

type Session interface {
	// value: a command attached inside a VM: its output, its input, and its
	// exit code once it ended; an error when the connection or ctx ended it
	// first.
	Stream
	Wait() (int, error)
}

type Listener interface {
	Accept() (Stream, error)
	Close() error
	Port() int
}

type Mac interface {
	// invariant: what the engine needs of the Mac it runs on: its clock, its
	// home, the files it reads, its state folder, the sockets it listens on,
	// and the children a former engine left behind.
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
	// value: waits until ready is closed, d of the clock at most.
	Await(ctx context.Context, ready <-chan struct{}, d time.Duration) error
	Home() string
	ReadFile(path string) ([]byte, error)
	// invariant: the last max bytes of a file, and whether the file held
	// more: the engine's log grows without bound, and is never read whole.
	Tail(path string, limit int64) (tail []byte, cut bool, err error)
	Exists(path string) bool
	MkdirAll(path string) error
	Remove(path string) error
	ListenUnix(path string) (Listener, error)
	// invariant: a TCP port the engine holds binds the loopback alone.
	ListenLoopback(port int) (Listener, error)
	Join(a, b Stream)
	Reap(dir string) int
}
