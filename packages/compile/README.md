# packages/compile

The spwn compiler. Full docs land in Commit 5 of Phase 1.

Short version: translates a provider-neutral spwn project into a
runtime-specific file tree (`Tree`). One `Runtime` implementation per
target (Claude Code, Codex, ...). Pure function: no I/O, no Docker.
