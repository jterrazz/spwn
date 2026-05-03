# Making codex a first-class spwn runtime — change audit

**TL;DR:** codex already has Tool + partial Spawn facets registered.
To promote it from "installable binary" to "interactive runtime on
par with claude-code" needs **7 code changes**, **2 catalog
additions**, and **3 test-infrastructure additions**. Estimated
effort: 1–2 days of focused work, end to end.

The biggest single piece of work is the **Render facet** (new
`codex/render.go`) and **architect's per-world runtime routing**
(today's Architect holds a single claude-code spawner; three entry
points still reach `a.runtime` instead of the per-world spawner).

## What already works

| Area | Status | Evidence |
|---|---|---|
| Adapter registration | ✅ | `packages/runtimes/codex/adapter.go` registers at init() |
| Tool (install recipe) | ✅ | `npm install -g @openai/codex` + `~/.codex/config.toml` seed |
| Credentials sync | ✅ (partial) | `/credentials/openai/auth.json` bind + PrelaunchShell symlink |
| Session-ID extraction | ✅ | `extractSessionID(runtimeName, line)` in talk.go already parses codex's `thread.started` / `thread_id` events |
| Runtime name resolution | ✅ | `source.ResolveRuntime` + `runtimeCanonical["spwn:codex"] → "codex"` |
| Build CLI flag | ✅ | `spwn build --runtime codex` resolves + invokes `transpile.Compile("codex", …)` — but fails because there is no registered codex renderer |
| World record persists runtime | ✅ | `models.World.Runtime` captured at spawn; talk.go reads it per-world |

## What's missing

Ordered by blast radius. Each section names the file(s) to touch
and the exact change.

---

### 1. `codex/render.go` — the renderer [BLOCKER]

**File to create:** `packages/runtimes/codex/render.go`

Mirror `claudecode/render.go`, but emit codex's native prompt
conventions. Codex reads `AGENTS.md` at the cwd on startup (the
convention OpenAI + several IDEs standardised on).

Output tree:
```
agents/<name>/AGENTS.md                  self-contained system prompt
agents/<name>/worlds/<id>/role.md        per-deployment role (optional — codex doesn't @-import)
```

Key design decisions codex's renderer needs:

- **Inlining vs imports.** Claude Code supports `@path/to/file.md`
  imports; codex does not (yet). So codex's AGENTS.md must inline
  everything the agent needs to see on startup: SOUL body, physics,
  faculties, roster, conventions, promoted playbooks, role. The
  claudecode adapter uses `@SOUL.md` / `@worlds/<id>/role.md` —
  codex must substitute and `cat` them into the final AGENTS.md.
- **`AGENTS.md` name collision.** The user writes a provider-neutral
  `AGENTS.md` as the source prompt (`source.ProjectSource.AgentSource.AgentMD`).
  The codex-rendered output file is ALSO `AGENTS.md`. Since the
  renderer emits into the compile tree under `agents/<name>/AGENTS.md`
  and the container runs codex with `cwd=/agents/<name>/`, there's
  no conflict — the source stays in `spwn/agents/<name>/AGENTS.md`
  on the host, the rendered version lands at
  `/agents/<name>/AGENTS.md` inside the container. Worth a comment
  in the doc though.
- **Worldbook reuse.** `worldbook.GeneratePhysics / GenerateFaculties /
  GenerateRoster` stay runtime-neutral. codex's renderer calls them
  and inlines the markdown. The only per-runtime concern is the
  outer wrapping (Conventions wording, inbox path format).
- **New helper:** `GenerateAgentAgentsMD(input AgentAgentsMDInput)`
  analogous to `claudecode.GenerateAgentCLAUDEMD`. Share the
  `demoteHeadings` / `stripLeadingH1` helpers — candidate extraction
  to `transpile/worldbook/shared.go`.

**Adapter wiring:** set `Render: Renderer` in
`codex/adapter.go`. `Register` in `runtimes/adapter.go` already
plumbs non-nil Render into `transpile.Register`.

---

### 2. Architect's global spawner → per-world spawner [BLOCKER]

**Files to touch:** `packages/architect/architect.go`,
`packages/architect/agent.go` (2 callsites),
`packages/architect/npc.go` (1 callsite),
`packages/architect/colony.go` (1 callsite).

Today `Architect` holds ONE `runtimes.Spawner` captured at
construction time:

```go
// architect.go:30
rt, err := runtimes.GetSpawner("claude-code")
// …
return &Architect{ …, runtime: rt }
```

And `agent.go:32`, `agent.go:95`, `npc.go:30` all call
`a.runtime.BuildCommand(…)`. That means even when a codex world is
deployed (image built correctly, container running codex), the
subsequent `spwn agent talk` / `SpawnAgentDetached` / `SpawnNPC`
ask the **claude-code** spawner to build the command — `claude
--dangerously-skip-permissions -p "…"` instead of `codex exec "…"`.

**Fix:**
1. Remove `runtime` field from `Architect` struct.
2. Resolve the spawner per-call from the world record:
   ```go
   u, err := a.rstate.Get(worldID)
   rt, err := runtimes.GetSpawner(u.Runtime) // falls back to claude-code via u.Runtime default
   cmd := rt.BuildCommand(…)
   ```
3. `colony.go:82` already has this pattern:
   ```go
   runtimeName := u.Runtime
   if runtimeName == "" { runtimeName = "claude-code" }
   ```
   Lift into a helper (`architect.resolveSpawner(u)` → `(runtimes.Spawner, error)`).

4. Mirror `apps/cli/agent/talk.go:76` which already does this
   lookup correctly — the architect-internal SpawnAgent paths just
   didn't.

---

### 3. `apps/cli/agent/talk.go` runtime-specific flag synthesis [BLOCKER]

**File:** `apps/cli/agent/talk.go` lines 93–98.

Hard-codes Claude Code output flags after BuildCommand:

```go
if message != "" {
    if talkOutputFormat == "stream-json" {
        runtimeCmd = append(runtimeCmd, "--output-format", "stream-json", "--verbose")
    } else {
        runtimeCmd = append(runtimeCmd, "--print", "--output-format", "json")
    }
}
```

These flags are claude-specific (`--print`, `--output-format`,
`--verbose`). codex's CLI takes different ones
(`codex exec --json "<prompt>"` or similar).

**Fix:** promote the flag synthesis into the Spawner interface.
Add to `runtimes.Spawner`:

```go
// OneShotFlags appends runtime-specific flags that tell the CLI
// to print + exit (vs. open interactive). Returns the modified
// slice. Optional — nil impl means no append.
OneShotFlags(base []string, outputFormat string) []string
```

Then:
- `claudecode.Spawner.OneShotFlags` appends `--print --output-format
  json` (or `stream-json --verbose`).
- `codex.Spawner.OneShotFlags` appends codex's `exec` subcommand and
  JSON flag (`exec --output-format json` or similar — verify in
  codex CLI docs).
- talk.go calls `rtSpawner.OneShotFlags(runtimeCmd, talkOutputFormat)`
  instead of the hard-coded append.

---

### 4. talk.go output parsing [BLOCKER]

**File:** `apps/cli/agent/talk.go` lines 215–231.

Non-streaming mode parses:
```go
var resp struct {
    Result    string `json:"result"`
    SessionID string `json:"session_id"`
}
```

claude-code's final envelope. codex's final JSON shape is different
(likely `output` field + `thread_id`). The fallback path uses
`extractSessionID(runtimeName, output)` which IS runtime-aware.

**Fix:** add to `runtimes.Spawner`:
```go
// ParseOneShotResult extracts the assistant's text body + session
// identifier from one raw CLI invocation output.
ParseOneShotResult(raw []byte) (text string, sessionID string, err error)
```

Implementations:
- claudecode: the existing `resp.Result` + `resp.SessionID` logic.
- codex: parse `thread.message` events or the final JSON envelope
  (thread_id + items).

talk.go replaces the inline json.Unmarshal with
`rtSpawner.ParseOneShotResult(output)`. The existing
`extractSessionID` fallback stays as a safety net.

---

### 5. Codex catalog surface (`spwn install codex`) [nice-to-have]

**Today:** `spwn install codex` works because `refs.ResolveCLI`
canonicalises `codex` → `spwn:codex`, and the runtime registry
exposes `spwn:codex` as a Tool via `RegisterDefaults`. ✅
(Already works — I tested this.)

**What's missing:** `spwn.yaml`-backed catalog entry with
title/tagline/description. Today claude-code is a Go-only tool
(no YAML entry) — it's invisible to `spwn install` catalog
browsing. Codex is the same. Both would benefit from an
(optional) catalog stub in `catalog/<runtime>/spwn.yaml` for
gallery discovery.

Not blocking for functionality; purely for `spwn install` UX
consistency.

---

### 6. Mock-codex for tests [infrastructure]

**Files to create:**
- `tests/fixtures/mock-codex/mock-codex.sh` (mirror mock-claude.sh)
- Update `tests/fixtures/Dockerfile.test` to ship `codex` stub too

The existing TS E2E suite relies on mock-claude for every spawn.
Adding codex coverage needs an analogous mock that records its
invocation + prints a plausible JSON envelope. Without this,
every codex E2E test would hit the real OpenAI API, unusable for
CI.

Same file count as mock-claude (~2 files, ~50 lines of bash).

---

### 7. Golden fixtures for codex renderer [infrastructure]

**Files to add under** `packages/runtimes/testdata/*/`:
alongside each existing `output_claude_code/` directory, add
`output_codex/` with the expected codex-rendered tree.

The golden test framework (`golden_test.go`) already walks every
`output_<runtime>_<something>` directory by prefix — adding codex
outputs is just populating those dirs once the renderer exists,
then `UPDATE_GOLDEN=1 go test ./packages/runtimes/...` regenerates.

Important scenarios to add output_codex/ for:
- `minimal-single-agent/` — sanity
- `colony-two-agents/` + `colony-three-agents-with-roles/` —
  multi-agent rendering
- `agent-with-core-profile/` + `agent-with-all-layers/` — SOUL
  inlining
- `world-with-explicit-workspaces/` + `world-without-knowledge/` —
  conditional paragraphs

Existing claude-code goldens are the template; the codex versions
differ in file name (AGENTS.md not CLAUDE.md) and in how imports
are handled (inlined not `@-referenced`).

---

### 8. Source-level naming collision: `AgentSource.AgentMD` vs codex output [design nit]

**File:** `packages/transpile/source/source.go`.

Today `AgentSource.AgentMD []byte` holds the bytes of the
USER-AUTHORED `spwn/agents/<name>/AGENTS.md`. The claude-code
renderer folds this into `CLAUDE.md`; codex's renderer should
fold it into the inlined `AGENTS.md` output.

Not a blocker — semantically clean (source file happens to share
a name with codex's output file, but they live at different paths:
`<projectRoot>/spwn/agents/<name>/AGENTS.md` vs
`<compile-tree>/agents/<name>/AGENTS.md`). Worth a comment on the
codex renderer calling this out so future maintainers don't get
confused.

---

### 9. `packages/architect/spawn.go` + destroy.go + npc.go — stepper messages [UX polish]

A few places say "Generating physics..." / "Spawning agent..." —
the language is runtime-neutral so this works for codex without
change. But comments in those files reference "claude" specifically
(`// Session management - claude-code is the only runtime.`).
Update comments to reflect the per-world resolution once §2 lands.

---

### 10. Scaffold template wording (`agent.yaml.tmpl`) [cosmetic]

**File:** `packages/project/internal/scaffold/templates/agent.yaml.tmpl`

Default scaffold hard-codes `runtime.backend: "spwn:claude-code"`.
Keep as default, but add a comment naming codex as a first-class
alternative:

```yaml
runtime:
  # One of: spwn:claude-code (default), spwn:codex
  backend: "spwn:claude-code"
```

---

## Mapping the work

| # | File(s) | Effort | Notes |
|---|---|---|---|
| 1 | `packages/runtimes/codex/render.go` (new) + share helpers | ~1 day | largest single piece; mirror claudecode/render.go |
| 2 | `architect/{architect,agent,npc,colony}.go` | ~2 hours | remove `Architect.runtime`, resolve per-world |
| 3 | `apps/cli/agent/talk.go` + `runtimes/Spawner` interface | ~2 hours | new `OneShotFlags` method |
| 4 | `apps/cli/agent/talk.go` output parser | ~2 hours | new `ParseOneShotResult` method |
| 5 | `catalog/codex/spwn.yaml` (new, optional) | ~30 min | title/tagline for gallery |
| 6 | `tests/fixtures/mock-codex/` (new) | ~1 hour | parallel to mock-claude |
| 7 | `packages/runtimes/testdata/*/output_codex/` | ~1 hour + review | UPDATE_GOLDEN regenerates |
| 8 | source.go doc comment | ~5 min | |
| 9 | architect/*.go doc comments | ~10 min | |
| 10 | scaffold template | ~5 min | |

**Total:** ~1–2 days of focused work, most of which is §1 (render) +
§6-7 (test infra).

## What spawning a codex world would look like after these changes

```bash
# Author:
cat > spwn.yaml <<'Y'
version: 1
name: my-project
worlds:
  default:
    agents: [neo]
Y

cat > spwn/agents/neo/agent.yaml <<'Y'
name: neo
description: Codex-backed agent.
runtime:
  backend: "spwn:codex"
dependencies:
  - "spwn:unix"
  - "spwn:codex"
Y

cat > spwn/agents/neo/SOUL.md <<'S'
# Neo
You speak in short, measured lines.
S

# Run:
spwn up                # image built with codex + spwn:unix; container up
spwn agent talk neo    # codex session opens; reads AGENTS.md from cwd
```

Inside the container on first `talk`, codex reads
`/agents/neo/AGENTS.md` (the rendered one) which inlines SOUL +
physics + faculties + roster + playbooks + role. The agent's voice
matches SOUL.md, conventions match the inlined section, etc — same
quality as the claude-code path.

## Risks

- **Codex CLI flag divergence** — OpenAI could change `codex exec`
  JSON shape. Mitigation: `ParseOneShotResult` is runtime-scoped so
  future drift is isolated to `codex/spawn.go`.
- **Rendering divergence tests** — easy to let the two runtimes'
  outputs drift (e.g. claude-code adds a new Convention, codex
  doesn't). Mitigation: move shared content into
  `worldbook.ConventionLines()` returning structured bullets, so
  both renderers consume the same source.
- **Session model differences** — claude's `--resume <id>` vs
  codex's `--thread <id>`. Already handled on the extraction side
  by `extractSessionID`; needs matching injection on the BuildCommand
  side.

## Related prior work

- Live-agent QA pass (commits `55c77630` through `3338f92c`) fixed
  5 claude-specific injection bugs that would ALSO affect codex:
  chown, bind-mount shadow, local-skills baking, hook execution,
  heredoc encoding. Codex benefits from every single one.
- Script-harness pass (commits `da978c1c`, `ad69422c`, `bc56a2ff`,
  `c42fb853`) added agent-name validation, cycle detection, `ls
  --json`, hook timeout. All runtime-agnostic.

## Recommended sequencing

1. **Day 1, morning:** refactor architect to per-world spawner (§2).
   Lands without user-visible change; unlocks later work.
2. **Day 1, afternoon:** add `OneShotFlags` + `ParseOneShotResult`
   to Spawner interface (§3 + §4). Claude-code impls are just
   refactors of today's inline logic. codex impls are stubs.
3. **Day 2, morning:** `codex/render.go` (§1). Worldbook helpers
   stay shared; adapter emits AGENTS.md.
4. **Day 2, afternoon:** mock-codex + goldens (§6, §7). Run
   `make test-ts` with codex scenarios added.

At end of day 2: `spwn up` on a codex-backed agent, `spwn agent talk
neo "hello"`, agent responds with SOUL-derived voice. Ship.
