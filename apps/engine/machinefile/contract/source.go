package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

type Source interface {
	// invariant: the source's catalog packages and Machinefiles, each by its
	// slash path under the source (`catalog/<category>/<name>.yaml`,
	// `machinefiles/<name>.yaml`), in path order, in the source's layer; a
	// source that holds neither folder answers no file.
	Read(ctx context.Context) (machinefile.Tree, error)
}

type Follows interface {
	// invariant: a source that follows a branch is read where its clone
	// stands; Pull moves the clone to the branch's newest commit, the one
	// gesture that changes what Read reads, and only forward: a newest
	// commit that does not descend from the clone's (a force-push, a
	// rewritten history, another default branch) is refused with a
	// *NotForwardError and the clone stays where it stood. It answers the
	// commit the clone stood at before, and the one it stands at after.
	Pull(ctx context.Context) (Pulled, error)
	// invariant: Back puts the clone at a commit it stood at, as Pull
	// answered it; the clone is then read there.
	Back(ctx context.Context, commit string) error
}

// value: the commit a clone stood at before a pull, and the one it stands
// at after; the same when the branch had not moved, or the pull failed.

type Pulled struct{ Was, Now string }

func (p Pulled) Moved() bool { return p.Was != p.Now }

// value: a pull refused because the branch's newest commit does not
// descend from the commit the clone is at: the source, both commits, and
// the clone's folder, whose removal is the owner's way to take the branch
// as it now is.

type NotForwardError struct {
	Source, At, Newest, Clone string
}

func (e *NotForwardError) Error() string {
	return "its branch is at " + short(e.Newest) + ", which does not follow " + short(e.At) + ", the commit its clone is at (a force-push, a rewritten history or another default branch): station keeps " + short(e.At)
}

func short(commit string) string { return commit[:min(7, len(commit))] }

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
