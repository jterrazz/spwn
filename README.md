<p align="center">
  <strong>spwn</strong>
</p>

<p align="center">
  Isolated Docker worlds where AI agents live, work, and evolve.
</p>

<p align="center">
  <a href="#quickstart"><strong>Quickstart</strong></a> &middot;
  <a href="https://spwn.sh/docs"><strong>Docs</strong></a> &middot;
  <a href="https://spwn.sh/manifesto"><strong>Manifesto</strong></a> &middot;
  <a href="CONTRIBUTING.md"><strong>Contributing</strong></a>
</p>

<p align="center">
  <a href="https://github.com/jterrazz/spwn/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT License" /></a>
  <a href="https://github.com/jterrazz/spwn/stargazers"><img src="https://img.shields.io/github/stars/jterrazz/spwn?style=flat" alt="Stars" /></a>
  <a href="https://github.com/jterrazz/spwn/releases"><img src="https://img.shields.io/github/v/release/jterrazz/spwn?style=flat" alt="Release" /></a>
</p>

<br/>

<p align="center">
  <img src="doc/app-screenshot.webp" alt="spwn — Worlds dashboard" width="720" />
</p>

<br/>

## What is Spwn?

Models mastered thinking. Spwn builds the missing half: **a reality for them to live in**.

Spwn is a CLI and desktop app that runs AI agents inside isolated Docker containers with persistent identity, multi-agent coordination, and physics-based security. You define the world. Your agents remember, adapt, and collaborate.

|        | Step            | Example                                                            |
| ------ | --------------- | ------------------------------------------------------------------ |
| **01** | Create an agent | `spwn agent new neo`                                               |
| **02** | Spawn a world   | `spwn up --agent neo -w ./my-project`                              |
| **03** | Watch it live   | Agent discovers tools, works on your code, remembers everything.   |

<br/>

## Quickstart

```bash
# Install (downloads latest release to ~/.local/bin)
curl -fsSL https://spwn.sh/install.sh | bash
```

```bash
# Create an agent and spawn a world
spwn agent new neo
spwn up --agent neo -w ./my-project --detach

# Talk to the agent
spwn agent talk neo "What is this project?"

# Check running worlds
spwn ls
```

Or build from source:

```bash
git clone https://github.com/jterrazz/spwn.git && cd spwn
make install
```

> **Requirements:** Go 1.25+, Docker

<br/>

## Features

<table>
<tr>
<td align="center" width="33%">
<h3>🌍 Isolated Worlds</h3>
Every agent runs in a Docker container with its own filesystem, compute, and network. Real constraints. Real physics.
</td>
<td align="center" width="33%">
<h3>🧠 Persistent Identity</h3>
Agents have a profile — persona, traits, purpose, skills, knowledge, playbooks. It survives across worlds.
</td>
<td align="center" width="33%">
<h3>🏗️ Multi-Agent Hierarchy</h3>
Leaders delegate to workers via inboxes. Workers report back. Flexible hierarchy, clear delegation.
</td>
</tr>
<tr>
<td align="center">
<h3>🔒 Physics-Based Security</h3>
No ACLs. If curl isn't installed, HTTP is impossible — not forbidden, physically absent. You can't prompt-inject a missing binary.
</td>
<td align="center">
<h3>🧬 Agent Evolution</h3>
Dream to learn, sleep to consolidate, fork to branch. Natural selection for agent behavior.
</td>
<td align="center">
<h3>🔌 Pluggable Runtimes</h3>
Claude Code, Codex, Aider, Pi — swap the thinking engine without touching your world config. 8 port interfaces, all swappable.
</td>
</tr>
</table>

<br/>

## CLI at a glance

```
spwn up --agent neo -w .                    Spawn a world
spwn ls                                     List active worlds
spwn agent talk neo "do this"               Talk to an agent
spwn agent dream neo                        Analyze experience, promote playbooks
spwn agent sleep neo                        Shutdown, consolidate knowledge
spwn agent fork neo neo-v2                  Clone an agent
spwn snap save <id>                         Snapshot a world
spwn architect start                        Always-on orchestration daemon
```

Full CLI reference → [`docs/cli/`](docs/cli/spwn.md)

<br/>

## Documentation

| Topic | Link |
|---|---|
| **Principles** — why spwn is built this way | [`docs/principles.md`](docs/principles.md) |
| **Agent identity** — profiles, memory, evolution | [`docs/identity.md`](docs/identity.md) |
| **World physics** — config, tools, constraints | [`docs/world-physics.md`](docs/world-physics.md) |
| **Tool catalog** — SDKs, runtimes, platform tools | [`docs/tool-catalog.md`](docs/tool-catalog.md) |
| **Architecture** — ports & adapters, module map | [`docs/architecture.md`](docs/architecture.md) |
| **Comparison** — vs LangChain, E2B, MCP, Docker | [`docs/comparison.md`](docs/comparison.md) |
| **CLI reference** — every command, auto-generated | [`docs/cli/`](docs/cli/spwn.md) |
| **Releasing** — release runbook | [`docs/releasing.md`](docs/releasing.md) |
| **Contributing** — setup, testing, conventions | [`CONTRIBUTING.md`](CONTRIBUTING.md) |

<br/>

## Community

- [Website](https://spwn.sh) &middot; [Docs](https://spwn.sh/docs) &middot; [Manifesto](https://spwn.sh/manifesto) &middot; [Issues](https://github.com/jterrazz/spwn/issues)

<br/>

## License

MIT © 2025 Spwn

---

<p align="center">
  <sub>Open source. Self-hosted. Built for people who want to give agents a world, not a wrapper.</sub>
</p>
