# ADR-009: One bootstrap verb, and no values in argv

**Status:** Accepted
**Date:** 2026-10-31

## Context

The first design had several visible bootstrap steps (`machines init`, a `zshrc` item, an `xdg-layout` item), a `skills` family beside `tools`, a visible `shell init zsh` and `completion`, and `config install --set item.key=value`. A value in argv lands in shell history, and a skill is installed content, not configuration.

## Decision

A skill is a catalogue tool with the method `skill`; there is no `skills` family. A config item that needs values asks them in a form on a terminal and is refused elsewhere with exit 2 and `Hint: run it on a terminal`; there is no `--set`. `station setup [--name <name>]` is the one visible, idempotent bootstrap verb: it registers this host, writes the managed `~/.zshrc` block and migrates `~/.studio/`, and `install.sh`, the image recipe and `make provision` call it. The plumbing the block evaluates, `station shell zsh` and `station completion`, is hidden. The Claude Code token is lendable, as `operator:claude:oauth-token`.

## Consequences

- A secret never travels in argv or shell history; a scripted install of an item with values is not possible.
- The command map has fewer families and one place to start.
- Hidden commands are documented as plumbing.
