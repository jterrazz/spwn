package calls

import (
	"context"
	"time"
)

type ThisMac interface {
	// value: what the machines service reads and changes on the Mac it runs
	// on, handed by wire over that Mac's own files and clock.
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
	Home() string
	UID() int
	User() string
	Hostname() string
	Exists(path string) bool
	// value: the names in dir, an error when it cannot be read.
	Names(dir string) ([]string, error)
	Remove(path string) error
	// value: one line added at the end of a file, made with its folder when it
	// is not; the file may be one another process appends to.
	Append(file, line string) error
}

type GuestRan struct {
	// value: what a command inside a VM said, as the engine relays it.
	Stdout, Stderr string
	ExitCode       int
}
