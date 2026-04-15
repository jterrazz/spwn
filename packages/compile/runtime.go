package compile

import (
	"fmt"
	"sort"

	"spwn.sh/packages/world/models"
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
// Phase 1 shape: just the data architect.Spawn already has on hand
// when it decides to render. Future phases will grow AgentSource /
// SkillSource / HookSource types loaded directly from disk so that
// Input can be built from a project path without a live Spawn call.
type Input struct {
	// Manifest is the parsed spwn.yaml (or whatever synthetic
	// manifest a single-agent spawn flow constructs).
	Manifest models.Manifest

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
}

// AgentInput is the per-agent slice of compile.Input.
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
