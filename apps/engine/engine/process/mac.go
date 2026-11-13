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
	Home() string
	ReadFile(path string) ([]byte, error)
	Exists(path string) bool
	MkdirAll(path string) error
	Remove(path string) error
	ListenUnix(path string) (Listener, error)
	// invariant: a TCP port the engine holds binds the loopback alone.
	ListenLoopback(port int) (Listener, error)
	Join(a, b Stream)
	Reap(dir string) int
}
