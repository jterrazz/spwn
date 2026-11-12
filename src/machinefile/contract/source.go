package contract

import (
	"context"

	"github.com/jterrazz/studio-station/src/machinefile"
)

type Source interface {
	// invariant: the source's catalog items and Machinefiles, each by its
	// slash path under the source (`catalog/<category>/<name>.yaml`,
	// `machinefiles/<name>.yaml`), in path order, in the source's layer; a
	// source that holds neither folder answers no file.
	Read(ctx context.Context) (machinefile.Tree, error)
}

type Follows interface {
	// invariant: a source that follows a branch is read where its clone
	// stands; Pull moves the clone to the branch's newest commit, the one
	// gesture that changes what Read reads.
	Pull(ctx context.Context) error
}

// value: a shared source that cannot be read: the clone, the fetch or
// the checkout failed.

type UnavailableError struct {
	Source string
	Cause  error
}

func (e *UnavailableError) Error() string {
	return "the source " + e.Source + " cannot be read: " + e.Cause.Error()
}

func (e *UnavailableError) Unwrap() error { return e.Cause }
