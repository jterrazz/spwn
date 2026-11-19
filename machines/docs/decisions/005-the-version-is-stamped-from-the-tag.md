# ADR-005: The version is stamped from the tag

**Status:** Accepted
**Date:** 2026-10-31

## Context

A Go module path without a `/vN` suffix ignores every tag at v2 or above, so build info alone would print a v1 pseudo-version on every release. Tag `v2.0.0` already exists, and the rename changes every command.

## Decision

The linker stamps the version from the tag: the Makefile uses `git describe --tags --always --dirty` and the publish workflow passes the tag. This release is `v3.0.0`. `station --version` prints the version and the commit.

## Consequences

- A release binary and a local build both say what they are.
- `go run` prints `dev`, by design.
- The ldflag is set in exactly two places.
