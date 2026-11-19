package calls

import (
	"context"
	"time"
)

type Machine interface {
	// value: what the apps service reads and keeps on the machine it runs
	// on, handed by wire over that machine's own files and commands.
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
	Home() string
	Glob(pattern string) []string
	LookPath(name string) (where string, found bool)
	Exists(path string) bool
	// value: name run with args within budget: its stdout then stderr and
	// its exit code; ran is false when it did not start.
	RunBriefly(ctx context.Context, name string, args []string, budget time.Duration) (output string, code int, ran bool)
	// value: as RunBriefly, with bin the child's whole PATH, so a command it
	// names is found in bin alone.
	RunSealed(ctx context.Context, bin, name string, args []string, budget time.Duration) (output string, code int, ran bool)
	// value: line appended to the log at path, its folder made.
	Keep(path, line string) error
}
