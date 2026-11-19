# ADR-059: The latest release is read by two calls: one reads what the engine kept, one asks GitHub

**Status:** Proposed
**Date:** 2026-11-19

## Context

The window shows whether this Mac runs the latest release of station and offers the installer when it does not. It needs one answer it can draw as given: the version the engine runs, the latest release, whether this Mac is behind, and when GitHub last answered.

Four shapes were weighed. A `Release` field on `engine.read` is required by its schema the day it lands, so a window newer than its engine gets an answer without it and cannot tell why; a new call is detected by the `calls` of the engine's hello. One read that may also fetch puts a call to a third party behind a read, against SC-09 and against the update check `TODO.md` designed, whose command "reads that cache with no network". An own call of the engine is not the Registry, so no story covers it and no schema judges its answer. The name `engine.release` is taken by the own call a guest releases a reserved port with.

A release made by the tag job carries its Linux binaries alone until the attach job adds the darwin ones ([Release](../guides/release.md)): a window that took it as the latest in that window of time would run an installer that fails.

## Decision

- **Two Registry calls of the engine family, both `in-engine`, applied where an engine runs.** `engine.read-release` answers what the engine kept and asks nothing of GitHub; `engine.refresh-release` asks GitHub now, keeps its answer, and answers the same shape. A GitHub that does not answer is `engine.release-unreachable`, kept as the last refusal beside the last answer.
- **The shape is `Installed`, `Latest`, `Behind`, `Head`, `Notes`, `Checked`, `Refused`.** `Behind` is `machines.Newer`, the one comparison of two releases, so `v0.21.0-alpha.1` is past `v0.20.0` and before `v0.21.0`; a version it cannot read (`dev`, `dry-run-<sha>`) is a build of no release, never behind. `Head` is a verdict the engine words and the window draws as given; the product's name in it is `catalog.ReleaseName`, one constant.
- **The latest release is GitHub's `/releases/latest` of `catalog.StationRepo`, taken only when it carries this Mac's binary** (`station-darwin-<arch>`); one that does not leaves the previous one.
- **What the engine kept is a file of the cache, `~/.cache/station/release.json` (`v: 1`)**, written by `engine.refresh-release` alone, so it outlives an engine restart and a later CLI line can read it with no network.
- **The engine asks every 6 h on a Mac, 30 min after a refused ask, and tells nothing**: its job invokes `engine.refresh-release` through the Router, so the gesture and the timer are one path; no guest and no VM asks GitHub.

## Consequences

- A window gates its page on `hello.calls` naming `engine.read-release`, and its gesture "read again now" is `engine.refresh-release`.
- The engine's launchd environment carries no token, so the asks are GitHub's anonymous ones: four a day per Mac, far under its 60 an hour; the adapter words a refusal as `GitHub said <status>` and does not parse the rate limit.
- The adapter is the engine family's own (`engine/adapter/github`), beside the apps' `githubrelease`: an adapter imports no other adapter, and the shared plumbing waits for the self-upgrade's lift of the download.
- The update check of `TODO.md` is amended, not replaced: its 12 h with jitter became 6 h, its `update.json` this `release.json`, and its stderr line, silences and preferences stay to build.
