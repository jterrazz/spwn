package transpile

import (
	"fmt"
	"sort"

)

// Runtime renders a project into a runtime-specific Tree. A Runtime
// is a pure function: no I/O, no Docker, no side effects. Output is
// deterministic for deterministic input.
//
// Each concrete runtime lives in its own sub-package under
// packages/compile/runtimes/ and registers itself via init().
type Runtime interface {
	// Name identifies the runtime (e.g. "claude-code", "codex").
	Name() string

	// Render translates the generic project into this runtime's
	// conventions. May inspect manifest + agents + skills + hooks.
	Render(input Input) (*Tree, error)
}

// Input is everything a Runtime needs to render a project. It exists
// so the signature can grow without breaking every runtime whenever
// a new field appears (profiles, hooks, per-agent overrides, ...).
//
// Today Input carries the minimal roster + manifest shape that
// every registered runtime needs. When a runtime wants richer data
// (AgentSource bodies, Skills, Hooks) we grow this struct rather
// than passing a second argument to Render.
type Input struct {
	// Manifest is the parsed spwn.yaml (or whatever synthetic
	// manifest a single-agent spawn flow constructs).
	Deps []string

	// VerifiedTools is the list of tool identifiers the runtime is
	// known to have access to inside the target container. The
	// faculties file lists these.
	VerifiedTools []string

	// WorldID is the id of the world being compiled. Some generated
	// files (roster, inbox paths) embed it directly.
	WorldID string

	// Agents is the roster of agents being deployed into this world.
	// Empty for NPC-only or boot-only renders.
	Agents []AgentInput

	// WorldKnowledgeMounted is true when the spawn pipeline bound a
	// host knowledge directory into /world/knowledge. Runtimes use
	// this to decide whether to emit the "knowledge base" boilerplate
	// in AGENTS.md / mind-management.md / per-agent CLAUDE.md. When
	// false, every reference to /world/knowledge/ is omitted — the
	// agent is never told a knowledge base exists.
	WorldKnowledgeMounted bool
}

// AgentInput is the per-agent slice of transpile.Input.
type AgentInput struct {
	Name string
	Role string
}

var runtimes = map[string]Runtime{}

// Register adds a Runtime to the global registry. Concrete runtime
// sub-packages call this from their init(). Re-registering the same
// name overwrites.
func Register(r Runtime) {
	runtimes[r.Name()] = r
}

func lookupRuntime(name string) (Runtime, error) {
	r, ok := runtimes[name]
	if !ok {
		return nil, fmt.Errorf("unknown runtime: %s", name)
	}
	return r, nil
}

// RegisteredRuntimes returns the sorted set of names currently
// registered via Register. Used by the CLI to surface an accurate
// "known runtimes" hint on typos, and by `spwn check` to warn when an
// agent declares a runtime that the catalog knows about but no
// compile adapter implements.
func RegisteredRuntimes() []string {
	out := make([]string, 0, len(runtimes))
	for name := range runtimes {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
