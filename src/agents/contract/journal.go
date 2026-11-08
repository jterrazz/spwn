package contract

import (
	"context"

	"github.com/jterrazz/studio-station/src/agents"
)

type Journal interface {
	// value: what the hook wrote of each session of a VM's agents, kept
	// inside the VM; nothing of it leaves the VM but the guest's report.
	//
	// invariant: every session kept, by session name; a session ended more
	// than catalog.AgentKeptFor ago is not read.
	Read(ctx context.Context) ([]agents.Agent, error)
	// invariant: writing a session replaces it, keeps its last
	// catalog.AgentStepsKept steps only, and leaves the other sessions; a
	// session whose name is not a plain name is refused; a write deletes the
	// sessions past their keep.
	Write(ctx context.Context, a agents.Agent) error
	// invariant: Forget removes a session, and forgetting one the journal
	// does not keep changes nothing.
	Forget(ctx context.Context, session string) error
	// invariant: Watch yields the journal as Read reads it, once at once and
	// again after each change, until ctx ends, then closes.
	Watch(ctx context.Context) (<-chan []agents.Agent, error)
}
