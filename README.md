<h1 align="center">spwn</h1>

<p align="center">
  AI agents orchestration as code. One source, run anywhere.
</p>

<p align="center">
  <a href="#quickstart"><strong>Quickstart</strong></a> &middot;
  <a href="docs/start/getting-started.md"><strong>Getting started</strong></a> &middot;
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
  <img src="docs/_assets/hero-v2.gif" alt="spwn - spawning an agent" width="560" />
</p>

<br/>

## Agents orchestration as code

The real power of AI isn't the model. It's the model _plus everything around it_. Oppenheimer in a chatbox can answer questions; Oppenheimer in a lab, surrounded by instruments, notebooks, colleagues, and years of memory, can change the world. **The environment is the multiplier.**

With spwn, **you build that environment, block by block**. Stack `spwn:python` with `spwn:qmd`, add a local `tool/ffmpeg`, pin a few skills and a soul, and your agent wakes up inside a sandbox assembled exactly for its job. Two agents in the same project can live in two totally different worlds: one talks to Postgres and runs tests, the other compiles video. Every block is a declarative file, reviewed in PRs, pinned in lockfiles, swapped like Lego.

Spawn it, commit it to git, ship it. If Terraform is infrastructure as code, spwn is **agents orchestration as code**: the same discipline, now for the minds that work on your repo. One `spwn build`, one portable artifact. **Docker for intelligence.**

<br/>

## Quickstart

```bash
curl -fsSL https://spwn.sh/install.sh | bash
```

Three commands. One working agent.

|        | Step               | Command          | What it does                                                                                                   |
| ------ | ------------------ | ---------------- | -------------------------------------------------------------------------------------------------------------- |
| **01** | Log in             | `spwn auth`      | Checks you're signed in to Claude Code (or any supported runtime).                                             |
| **02** | Scaffold a project | `spwn init`      | Drops `spwn.yaml` + a starter `neo` agent into the current directory.                                          |
| **03** | Talk to your agent | `spwn agent neo` | Opens an interactive session with neo inside a sandboxed Docker world. Container lifecycle is handled for you. |

Prefer a bundled demo? `spwn init matrix` drops a ready-made multi-agent world into the current directory (swap `matrix` for any template slug in the [catalog](catalog/)).

What lands in your project, and the four ideas behind it, is [Getting started](docs/start/getting-started.md).

> **Requirements:** Docker

<br/>

## Features

- **Agents as Code.** Commit agents alongside your app, review behavior changes in PRs, ship the same mind to every machine.
- **Composable environment.** Stack `spwn:python`, `spwn:qmd`, `tool/ffmpeg`, whatever your agent needs. Each one wakes up in a sandbox assembled for its job.
- **Reproducible.** One `spwn build` produces a Docker image, or a runtime-native tree (claude-code, codex and the others) if you skip Docker. Byte-identical on every machine.
- **Worlds.** Bundle agents, workspaces, and knowledge into a world. `spwn up` deploys them together, `spwn down` tears it all down.
- **Persistent.** Memory is a folder of markdown files. Readable, diffable, and alive across restarts.
- **Checked.** `spwn check` walks your project and surfaces bad refs, missing files, or lockfile drift before spawn.
- **Automations.** Wake an agent on a cron tick or a filesystem event. `worlds.<name>.automations` in spwn.yaml; receipts in `.spwn/runs.jsonl`; `spwn automation daemon` runs the engine. [Full guide](docs/guides/automations.md).

> _"The next breakthrough isn't smarter models. It's richer worlds."_

<br/>

## Learn more

- [Getting started](docs/start/getting-started.md): the four ideas behind spwn, an agent's files, a world, and how one source reaches every runtime.
- [Use cases](docs/start/use-cases.md): a scientist composed from blocks, an agent shipped with its repo, a forked mind, a sealed room for untrusted code.
- [The manual](docs/README.md): guides, concepts and the reference, from the [primitives](docs/reference/primitives.md) to [every command](docs/reference/commands.md).
- [Contributing](CONTRIBUTING.md): set-up, tests and conventions.

<br/>

## Community

- [Website](https://spwn.sh) &middot; [Docs](https://spwn.sh/docs) &middot; [Manifesto](https://spwn.sh/manifesto) &middot; [Issues](https://github.com/jterrazz/spwn/issues)

---

<p align="center">
  <sub>Open source. Self-hosted. Built for people who want to give agents a world, not a wrapper.</sub>
</p>
