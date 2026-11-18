package calls

import (
	"context"
	"time"
)

type Machine interface {
	// value: what the disk service reads and changes on the machine it runs
	// on beyond its DiskUsage, handed by wire over that machine's own files
	// and commands.
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
	Glob(pattern string) []string
	Exists(path string) bool
	// value: the names in dir, an error when it cannot be read.
	Names(dir string) ([]string, error)
	ReadFile(path string) ([]byte, error)
	// value: dir removed when it is empty, else left.
	Prune(dir string)
	// value: whether name is a command on PATH.
	Has(name string) bool
	// value: name run with args and env, its output on stderr; Found is
	// false when there is no such command.
	Run(ctx context.Context, name string, args, env []string) (Ran, error)
	// value: data written to path in one rename, its folder made.
	Keep(path string, data []byte) error
}

type Ran struct {
	Found      bool
	Code       int
	LastStderr string
}

type ContainerSpace interface {
	// value: the bytes a container runtime holds, -1 when it did not answer
	// in time, 0 when it holds none or is stopped.
	Bytes(ctx context.Context) int64
}
