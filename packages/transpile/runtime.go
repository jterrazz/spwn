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
	// this to decide whether to mention the "knowledge base" in the
	// per-agent CLAUDE.md's Roster / Conventions sections. When
	// false, every reference to /world/knowledge/ is omitted — the
	// agent is never told a knowledge base exists.
	WorldKnowledgeMounted bool

	// Skills is every skill the renderer should materialise into each
	// agent's native skill directory (`.claude/skills/<n>/` for
	// claude-code, `.agents/skills/<n>/` for codex). Union of two
	// sources:
	//   - user-authored under spwn/skills/<n>/SKILL.md (loaded by
	//     packages/transpile/source)
	//   - tool-shipped skills resolved from each agent's `deps:`
	//     (collected by the architect at spawn time via the
	//     dependency resolver and handed in here)
	// The renderer treats both kinds identically; the single source
	// of truth for what "a skill" is lives in SkillEntry.
	Skills []SkillEntry
}

// SkillEntry is one complete skill the renderer must emit into each
// agent's native skill directory. Files is keyed by path relative to
// the skill's own root (`SKILL.md`, optional sidecar files like
// `template.md`, `scripts/run.sh`). SKILL.md is always required —
// both Claude Code and Codex refuse to load a skill without it.
type SkillEntry struct {
	Name  string
	Files map[string][]byte
}

// AgentInput is the per-agent slice of transpile.Input.
type AgentInput struct {
	Name string
	Role string

	// Soul is the raw bytes of spwn/agents/<name>/SOUL.md. Renderers
	// that can @-import (claude-code) ignore this; renderers that
	// must inline everything into a single boot prompt (codex) read
	// it as the agent's identity body. Nil when SOUL.md is missing —
	// renderers that rely on it treat a nil Soul as "agent has no
	// declared identity".
	Soul []byte
	// AgentMD is the raw bytes of spwn/agents/<name>/AGENTS.md — the
	// user-authored provider-neutral prompt body. Renderers that
	// consume it inline it at the bottom of the per-agent boot prompt
	// (codex) so the user's own instructions survive transpilation.
	// Claude-code ignores this because it ships the file to
	// /agents/<name>/AGENTS.md via docker-cp and the CLAUDE.md entry
	// point doesn't reference it.
	AgentMD []byte

	// Playbooks lists the agent's promotable playbooks — the subset of
	// files under spwn/agents/<name>/playbooks/ that carry valid
	// `name:` + `description:` frontmatter. The renderer emits the list
	// as a discoverability index in the entry file (CLAUDE.md for
	// claude-code, AGENTS.md for codex) so the runtime sees which
	// playbooks are available as named shortcuts. Plain playbooks
	// without frontmatter stay invisible until the agent decides to
	// promote them.
	Playbooks []PlaybookEntry
}

// PlaybookEntry is one frontmatter-promoted playbook, ready to index
// in CLAUDE.md. Name comes from frontmatter `name:` (not the
// filename) so agents can rename a playbook without touching the
// file on disk. Description is the one-line `description:` that
// explains when to reach for this procedure.
type PlaybookEntry struct {
	Name        string
	Description string
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
