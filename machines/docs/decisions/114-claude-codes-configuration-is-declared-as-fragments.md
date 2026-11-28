# ADR-114: Claude Code's configuration is declared as fragments

**Status:** Proposed
**Date:** 2026-11-28

A machine made from a Machinefile should reach a working Claude Code with no gesture from its owner but signing in: the hooks its agents rely on, the apps those hooks run, and nothing that phones home unasked. This record decides which of Claude Code's files Machines writes into, what it writes, what it never touches, and what is left to a person. The merge itself is [ADR-110](110-a-fragment-is-merged-into-a-file-the-owner-keeps-and-taken-back-by-its-record.md)'s; the packages are [Settings](../reference/settings.md#the-packages)'.

## Context

Claude Code reads its user settings from `~/.claude/settings.json`, strict JSON, and writes that file itself when its owner saves a setting ([Claude Code settings](https://code.claude.com/docs/en/settings.md)). Its other file, `~/.claude.json`, holds the account, the MCP servers of user scope and per-project state, and Claude Code rewrites it while it runs ([Claude Code MCP](https://code.claude.com/docs/en/mcp.md)). Hooks are a list per event in `settings.json`, each group a matcher and its handlers, and entries from every level are merged ([Claude Code hooks](https://code.claude.com/docs/en/hooks.md)).

The apps that hook into Claude Code each ship an installer that writes these files, and each installer also offers to route Claude Code's requests through a local proxy of its own (`env.ANTHROPIC_BASE_URL`): rtk's `rtk init -g` at v1.2.1, caveman's `caveman setup` at v4.2.0. caveman's CLI also sends usage telemetry unless its choice is recorded off.

## Decision

- **Only `~/.claude/settings.json`, and only by merge.** A Claude Code hook is a fragment of that file, merged by its record, under the setting of the app whose command it runs, which `requires` that app and is proposed by the app's `then`. The fragment is the exact entry the app's own installer writes, read from the source at a pinned tag, never written from its docs.
- **`rtk-claude-hook`** merges the one entry `rtk init -g` adds at v1.2.1: on `PreToolUse`, matcher `Bash`, the command `rtk hook claude`. Nothing else of that init is taken: not `RTK.md` and its `@RTK.md` line in `~/.claude/CLAUDE.md`, not the `filters.toml` template, not the backup, and never the proxy.
- **`caveman-cli`** installs the caveman CLI from npm on macOS and from bun on Linux, and proposes **`caveman-telemetry-off`**, which merges `{"telemetry": {"enabled": false, "decidedAt", "promptVersion": 6}}` into `~/.caveman/cloud.json`, the choice the CLI reads before every send; recorded before its first run, the CLI never mints its default, telemetry on. A choice the owner recorded already is his and kept. caveman's Claude Code hooks are not declared: they are nine whole files in `~/.claude/hooks/` beside their entries in `settings.json`, and a setting's mode is one per method, so the files and their entries cannot be one setting, and two settings could land an entry without its file, ADR-052's failure. The proxy is never taken.
- **No baseline of Claude Code keys.** Every documented key that changes Claude Code's behaviour (model, permissions, attribution, retention, Claude Code's own telemetry) is its owner's choice; `$schema` changes nothing but an editor's completion. A Machinefile that wants one declares its own fragment.
- **No MCP server is declared.** A server of user scope lives only in `~/.claude.json`, a file Claude Code rewrites as it runs, which a merge would race; `claude mcp add --scope user` writes the same file, needs Claude Code installed and signed in, and has no record off could take back. An MCP app stays an app (`playwright-mcp`), its server added by its owner.
- **Never touched:** `~/.claude.json`, the account and its sign-in, permissions, the model, `CLAUDE.md` and every Markdown file of `~/.claude/`.

## Consequences

- A Machinefile that lists `rtk` gets its hook on the first apply, and one that lists `caveman-cli` gets caveman with telemetry off, before any session runs; signing in to Claude Code stays a person's gesture.
- Off takes back the entry and the values Machines merged, and nothing it did not write. It cannot take back what the apps write themselves afterwards: a later `rtk init -g` sees the entry and adds none, but writes `RTK.md`, the `@RTK.md` line and the backup; a caveman run writes its own keys (`anonymousId`, its account) beside the choice, and once off has taken the choice back, the next interactive caveman run mints its default, telemetry on.
- Claude Code may rewrite `settings.json` between two applies; the merge reads the file it replaces and keeps what it finds, so an entry Claude Code dropped is merged again on the next apply.
- The npm and bun methods carry no version, so the catalog cannot pin caveman's CLI: the facts above are v4.2.0's, and a Machinefile pins a version with a range, bun's only.
- Left open: per-copy merge modes, which would let caveman's hooks be one setting; an AI category of settings, which would move the wire's closed list; rtk's awareness text as a whole file under `~/.claude/rules/`; and caveman's agent skill, which is its own package.
- Not proven: no Claude Code session has read these settings yet; the first machine made from such a Machinefile will.
