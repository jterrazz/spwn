package calls

import (
	"context"
	"time"
)

type Machine interface {
	// value: what setup reads and changes on the machine it runs on, handed
	// by wire over that machine's own files and commands.
	Home() string
	User() string
	Hostname() string
	Sleep(ctx context.Context, d time.Duration) error
	// value: the bytes at path; found is false when there is no such file.
	ReadFile(path string) (data []byte, found bool, err error)
	WriteFile(path string, data []byte, mode uint32) error
	MkdirAll(path string, mode uint32) error
	// value: the permission bits of path, or otherwise when there is none.
	Mode(path string, otherwise uint32) uint32
	Exists(path string) bool
	// value: whether path is a folder; found is false when there is none.
	IsDir(path string) (dir, found bool)
	// value: the entries of dir; found is false when there is no such folder.
	Entries(dir string) (entries []Entry, found bool, err error)
	Glob(pattern string) []string
	// value: removed is false, and err nil, when there was nothing at path.
	Remove(path string) (removed bool, err error)
	// value: crossDevice is true when from and to are on two volumes.
	Rename(from, to string) (crossDevice bool, err error)
	// value: name run with args within budget: its stdout and its exit code.
	Output(ctx context.Context, name string, args []string, budget time.Duration) (stdout string, code int, err error)
	// value: name started detached with env, its output appended to log.
	Start(name string, args, env []string, log string) error
	Alive(pid int) bool
	Terminate(pid int) error
	Kill(pid int) error
}

type Entry struct {
	Name string
	Dir  bool
}
