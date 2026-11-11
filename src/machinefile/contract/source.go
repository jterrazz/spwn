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

// value: a shared source that cannot be read at its commit: the clone or
// the checkout failed.

type UnavailableError struct {
	Source, Commit string
	Cause          error
}

func (e *UnavailableError) Error() string {
	return "the source " + e.Source + " cannot be read at " + e.Commit + ": " + e.Cause.Error()
}

func (e *UnavailableError) Unwrap() error { return e.Cause }
