package calls

import (
	"context"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/journal/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

type Deps struct {
	Journal contract.Journal
	Now     func() time.Time
	Macs    AllMacs
	// value: whether the catalog still knows an app, a setting, a skill or
	// an image a line names.
	Known func(journal.Subject) bool
}

type AllMacs interface {
	// value: every Mac and its VMs, as This Mac lists them.
	Machines(ctx context.Context) (machines.AllMachines, error)
	// invariant: each Mac's journal.kept through the Router, a few at a
	// time, each within its budget (remote/each): a Mac that does not
	// answer in time is its own answer's Why, never the others' delay.
	Kept(ctx context.Context, macs []string, in KeptIn) map[string]Answer
}

type Answer struct {
	// value: one Mac's lines, or why it gave none, in the owner's words.
	Lines []journal.Line
	Why   string
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(Kept, d), call.Bind(Read, d)}
}
