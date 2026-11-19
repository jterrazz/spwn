# ADR-003: mise owns the runtimes

**Status:** Accepted
**Date:** 2026-10-31

## Context

Language runtimes came from nvm, per-runtime brew installs and a hand-set `JAVA_HOME`, each with its own shell lines.

## Decision

mise manages node, go, python, bun, rust and java on every machine. `station tools install node` runs `mise use --global`, and the shell block activates mise. nvm, openjdk and the `java` item leave the catalogue. A runtime installed elsewhere shows as degraded, "on PATH, not managed by mise", and the repair is the install command. Station never edits shell lines outside its own block.

## Consequences

- One mechanism per runtime, on a host and in a Linux VM alike.
- A machine with nvm needs one command per runtime and a hand removal of its old lines.
- npm and uv bundle entries run with mise's shims on PATH.
