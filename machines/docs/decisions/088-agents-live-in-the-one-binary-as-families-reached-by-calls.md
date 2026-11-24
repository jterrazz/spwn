# ADR-088: Agents live in the one binary, as families reached by calls

**Status:** Accepted
**Date:** 2026-11-24

Supersedes [ADR-052](052-station-keeps-no-agents.md).

Accepted by the owner on 2026-11-24, on the owner's word.

## Context

ADR-052 took every notion of a running agent out of the engine and gave it to "another binary": Agents' `spwn`, built from `apps/cli` at the repository root. The owner then chose one program for everything (the repository's [ADR-021](../../../docs/decisions/021-one-program-spwn-one-command-for-everything.md)). Asked whether one command should run over two executables or be one linked binary, he answered « un seul programme » (2026-11-24); in the audit of the same day: « Spwn command unique pour tout tout tout ».

What the two trees hold today decides how:

- A call already runs wherever its machine is. The far side returns data and this binary renders it with its own pages ([ADR-016](016-api-v2-and-remote-by-api.md)); before 1.0 every machine runs the same release ([ADR-072](072-before-1-0-every-station-runs-the-same-release.md)).
- The kit needs no mode for a command that runs here: `kit.Leaf.Run` takes an `IO` and may do anything. A leaf is tied to the engine only when it invokes a call.
- Agents' modules that work on files (`agent`, `activity`, `platform`, `project`, `dependency`, `transpile`, `runtimes`, `automation`) import no world, container or compile package. Only four files of Agents import the Docker client, and they leave with the world.
- The engine already drives an outside engine through its command line (tart) and already imports a sibling module through a `replace` line (`catalog`).
- Merging the two `go.work` files would take the Go packages out of Machines' task graph, and moving Agents' modules under `machines/` would put them under the engine's lint.

A probe linked the eight modules into the linux binary on a throwaway branch (P3, 2026-11-24): the binary grew 1.5 %, and none of the eight brought a Docker, OpenTelemetry or gRPC dependency.

## Decision

- **Agents' nouns become engine families.** `agents`, `teams`, `skills`, `automations` and `project` each get a family under `apps/engine/` (`model`, `contract`, `calls`, `adapter/fake` and one real adapter) and leaves under `apps/engine/surface/cli/`, on the kit, as every family already has.
- **Every Agent act is a call** in `remote/api/calls.json`, with its own `Where`: `agents.*` at the target, `project.*` in this process, since a project is the folder where the command runs. `--json`, `spwn.error/v1` and the exit table are the kit's, and Spwn.app reads the same calls through the types `make generate` projects. `apps/api` has no successor.
- **`--on` works by construction.** The call is routed by `Env.Target()`, the far `spwn` runs it, this binary renders it, and its `--json` is the far one byte for byte. `project` refuses `--on`.
- **The engine requires Agents' modules; no workspace is merged.** `apps/engine/go.mod` requires the `spwn.sh/packages/*` modules it uses at `v0.0.0`, with one `replace` line for each `spwn.sh/` module in the graph. The root `go.work` and Agents' modules move to the engine's Go version in the same commit, so the code shipped is tested by its own toolchain. Each module is judged once, by the gate of the tree it lives in.
- **Only an adapter imports an Agents package.** A layers rule allows a `spwn.sh/` import in an adapter row alone.
- **The binary links no Docker client.** `TestTheBinaryLinksNoDockerClient`, beside `TestTheReleaseBinaryLinksNoWorld`, fails on any `github.com/moby/` or `github.com/docker/` in the darwin and linux binaries' dependencies. Docker, if it survives, is driven through its command line, as tart is.
- **The same binary goes into every VM**, the Agent families included, since an agent runs on its machine. No build tag makes a VM's binary a different program.
- **What a machine installs stays a package**, as ADR-052 had it: the harnesses (`claude`, `codex`) and the agent skills are catalog apps. Herdr is a package too, and it now holds the agents' terminals: [ADR-089](089-an-agent-runs-in-a-herdr-terminal-on-its-own-machine.md).
- **No compatibility.** Each Agent command is deleted from `apps/cli` in the lot that brings it here, so a command exists once at every step, and nothing is kept for a client of the old binary.

## Consequences

- ADR-052 is superseded. Its "another binary owns agents" no longer holds; its "a machine installs packages" and "no compatibility" are carried here; its "Herdr stays a package, never station's wiring" is reversed for the agents' terminals by ADR-089.
- `apps/cli` and the root `specs/cli/` are deleted once the last Agent command lives here. Until then the root gate keeps judging the domain modules.
- Two YAML libraries are linked, `go.yaml.in/yaml/v3` and `gopkg.in/yaml.v3`: the same code twice, unified by a later change.
- Agents' words have to pass the vocabulary guard, and every new page owes its four fixtures.
- The Mind stays in `~/.spwn`, outside [ADR-007](007-files-live-in-the-xdg-folders.md)'s folders, until the rename moves it.
