package store

import (
	"sync/atomic"
	"time"
)

var writes atomic.Uint64

func Wrote() {
	// invariant: every store that writes a file of this process calls Wrote
	// after the write, so a store holding what it read of that file knows,
	// whatever the clock of the disk says, that another store of this process
	// may have changed it; a write of another process shows in the file's Stamp.
	writes.Add(1)
}

func Written() uint64 { return writes.Load() }

type Stamp struct {
	// value: what a file's stat says of its content: when it was modified
	// and how long it is, which a write of another process changes.
	Mod  time.Time
	Size int64
}

type Info interface {
	ModTime() time.Time
	Size() int64
}

func StampOf(info Info) Stamp { return Stamp{Mod: info.ModTime(), Size: info.Size()} }
