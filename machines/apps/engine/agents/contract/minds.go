package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/agents"
)

type Minds interface {
	// invariant: every method acts on the agents of one project, its root
	// an absolute path on this machine; an empty project is the user's own
	// agents. A name no agent has is agents.ErrNotFound, a name taken
	// agents.ErrExists.

	// invariant: every agent of the project, by name.
	List(ctx context.Context, project string) ([]agents.Agent, error)
	// invariant: the agent, its layers by name, each layer's files by name.
	Mind(ctx context.Context, project, name string) (agents.Mind, error)
	// invariant: a new Mind with its layers empty, its agent
	// agents.Blank(name) with e applied; name is valid and e passes e.Check.
	Create(ctx context.Context, project, name string, e agents.Edit) (agents.Mind, error)
	// invariant: the agent as the edit leaves it; e passes e.Check.
	Edit(ctx context.Context, project, name string, e agents.Edit) (agents.Agent, error)
	// invariant: the agent's Mind is gone from its machine, every layer of it.
	Delete(ctx context.Context, project, name string) error
	// invariant: a new agent named to, with a copy of every layer of from.
	Fork(ctx context.Context, project, from, to string) (agents.Forked, error)
	// invariant: one archive of the Mind written into folder, named
	// <name>agents.ArchiveSuffix; the Mind is left as it is.
	Export(ctx context.Context, project, name, folder string) (agents.Exported, error)
	// invariant: a new agent named as from the archive's Mind; an archive
	// that does not read is agents.ErrBadArchive and leaves no agent behind.
	Import(ctx context.Context, project, archive, as string) (agents.Mind, error)
	Dream(ctx context.Context, project, name string) (agents.Dreamed, error)
	// invariant: the Mind's stale playbooks are archived, its old sessions
	// pruned, and a sleep log of the day is written into its journal layer.
	Sleep(ctx context.Context, project, name string) (agents.Slept, error)
}
