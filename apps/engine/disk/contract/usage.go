package contract

import "context"

type DiskUsage interface {
	// invariant: each root answers once, its bytes on its own volume (a
	// sparse file counts its blocks); a root past ctx answers -1.
	Sizes(ctx context.Context, roots []string, each func(i int, bytes int64))
	// invariant: the volume the owner's data lives on.
	Volume(ctx context.Context) (total, free int64, err error)
	// invariant: the folder is at to afterwards and nothing is left at from,
	// by one rename: a destination that holds anything, or one on another
	// volume, is refused and nothing moves.
	Move(ctx context.Context, from, to string) error
	// invariant: every path is removed, a read-only tree included; the first
	// failure is the one returned.
	Erase(ctx context.Context, paths []string) error
	// invariant: whether a process of that exact name runs, so nothing is
	// erased under a running app.
	Running(ctx context.Context, process string) bool
}
