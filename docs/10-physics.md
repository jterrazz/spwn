# Physics

Every world has its own physics: the constants, laws, and elements that decide what is possible inside it. This chapter is what those three mean, what the agent reads at startup, and why capability is modelled as physics rather than as configuration. How they are declared is [Primitives](07-primitives.md).

## The three pillars

- **Constants** are finite resource limits — CPU, memory, disk, timeout. They are fixed for the world's lifetime.
- **Laws** are rules the environment enforces — network policy, process limits, filesystem mounts. They cannot be circumvented from inside, only declared differently in the next world.
- **Elements** are the building blocks that exist — the tools and capabilities the composition declares. If an element is not declared, nothing in the world can be made from it.

## Input and output

The manifests are prescriptive: what reality should be. What the agent reads is descriptive: what reality is after the build.

- **physics** — the constants, laws, and topology of its world.
- **faculties** — the verified elements and bridges: what it can actually do.

Both are rendered as plain markdown into the world context at build time, so any runtime that reads files understands its reality without a custom parser or an injected system prompt. Declared elements are verified during the image build — each tool's `verify:` probes run before any agent wakes up — and only what is proven present reaches the faculties.

## The operating manual

Every agent wakes up with an entry file assembled at build time from the world's shared context (physics, faculties, roster) plus its own identity and role. The build renders it to the convention of each runtime — `CLAUDE.md` for Claude Code, `AGENTS.md` for codex-style runtimes — so the runtime boots fully loaded with nothing to chase at startup.

Platform instructions travel the same way. Rather than teaching every runtime adapter how spwn works, the build ships standardised markdown into every world; any runtime that reads files bootstraps itself from it. Two kinds of skill coexist, and the split is the point:

|          | System skills            | Agent skills                 |
| -------- | ------------------------ | ---------------------------- |
| Source   | The platform             | The agent's composition      |
| Optional | No, always present       | Yes, per agent via `skill/…` |
| Mutable  | No, read-only in a world | Yes, authored and evolved    |
| Purpose  | Platform operations      | Domain capabilities          |

## Elements bridged from the host

Some elements are not binaries in the image but host capabilities surfaced inside the world as ordinary commands by the [gate](08-gate.md). From the agent's side there is no difference: a bridged capability is an element that exists, listed in its faculties. A capability that is not bridged appears nowhere — not forbidden, absent.

## Why elements are physics, not configuration

The usual approach hands the agent a tool client and a list of servers. The agent then knows it is calling external services and can be talked into calling unauthorised ones. Modelling capability as physics removes the target:

- **The bridge is invisible.** The agent sees a command, not an endpoint.
- **Undeclared elements do not exist.** There is nothing to call, manipulate, or jailbreak.
- **The faculties are the single source of truth.** What the agent reads is exactly what is available — no hidden capability, no surprise restriction.

The layers stack: a bridged capability needs both the law-level gate connection and the element-level command, so even a binary that existed would have no host service to reach.

## Open question: timeout behaviour

Undecided — what happens when a world hits its timeout constant: hard kill (SIGKILL, no cleanup), graceful shutdown (SIGTERM, wait, SIGKILL, destroy), extend-and-warn (notify the agent, extend once, then shut down gracefully), or pause (freeze the world and wait for the operator). Leaning toward graceful shutdown with a configurable grace period.

## Related

- [Worlds](09-worlds.md) — the container the physics apply to.
- [Primitives](07-primitives.md) — the manifests that declare constants, laws, and elements.
- [Gate](08-gate.md) — how a host capability becomes an element.
