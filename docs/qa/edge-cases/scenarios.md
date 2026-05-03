# spwn — Edge-Case & Failure-Mode Scenario Catalog

These are the **complicated** scenarios the prior two suites
(CLI-harness + agent-behavior) miss by design. They probe:

- **Concurrency** — two processes fighting over the same state.
- **Partial failure** — spwn aborts mid-pipeline; what's left behind?
- **State-machine holes** — world in unexpected / impossible states.
- **Filesystem edges** — special chars, permissions, large files,
  symlinks, binary content, case-insensitive FS.
- **Tool install failures** — exits non-zero mid-build, network
  drops, produces wrong output, two tools colliding.
- **Skill/playbook conflicts** — same name across sources, malformed
  frontmatter variants, @-import chains.
- **Hook misbehavior** — hooks that hang, mutate state, exit
  non-zero, or try to do too much.
- **Auth** — missing/expired/wrong-provider credentials.
- **CLI UX** — TTY vs pipe, SIGINT, partial JSON, completion.
- **Resource limits** — PidsLimit, disk, memory.
- **Snapshot/restore** — cross-version, overlapping IDs, many snaps.
- **Multi-agent** — name collision with system reserved, inbox
  overflow, circular messages.

Each scenario: **Hypothesis** (expected behavior) · **Probe** (one-shot
command or setup) · **Pass criteria** · **Failure mode** (what a bug
looks like). Every scenario is executable without a human.

Legend: ✅ ran + passed · ❌ ran + failed (with fix commit) · 🟡 ran,
documented caveat · ⚪ not run.

---

## A. Concurrency (1-8)

### A1. Two `spwn up` processes racing on the same project
**Hypothesis:** one wins, the other fails cleanly; no zombie
containers; state.json not corrupted.
**Probe:** launch two `spwn up` in background, wait both, count
containers + check `spwn ls` state, re-run `spwn check --json`.
**Failure:** 2 containers for 1 world, or state.json parse error.

### A2. `spwn down` mid-spawn
**Hypothesis:** if down races ahead, spawn either completes or fails
cleanly — no orphan container.
**Probe:** `spwn up &` then `spwn down` ~1s later; after both return,
`docker ps` should be clean.

### A3. `spwn up` after SIGKILL'd prior spawn
**Hypothesis:** next `spwn up` detects stale state.json + half-
created container; cleans up and proceeds.
**Probe:** kill spwn up mid-flight (`pkill -9 spwn`), inspect state,
re-run up.

### A4. Concurrent `spwn agent talk` to same agent
**Hypothesis:** both sessions work (claude's session-id machinery
serialises them), or one errors cleanly.
**Probe:** two simultaneous `spwn agent talk neo "ping"`.

### A5. Two projects spawning concurrently with same world config name
**Hypothesis:** isolated — each has its own container with unique
world-id.
**Probe:** /tmp/p1 + /tmp/p2 both `spwn init && spwn up`.

### A6. `spwn build` while `spwn up` runs
**Hypothesis:** both read the project tree independently; neither
corrupts the other.
**Probe:** `spwn up &; spwn build --tree-only`.

### A7. Agent `docker cp`'d while container is being destroyed
**Hypothesis:** SyncOut during destroy is atomic; if it fails
(container already gone), warnings but no panic.
**Probe:** `spwn down &` then observe logs.

### A8. Two agents writing the same `/world/knowledge/<file>` concurrently
**Hypothesis:** last-write-wins on host (bind mount is a real FS).
No lockfile, so races exist; should not corrupt.
**Probe:** two agents `echo A > /world/knowledge/same.md` vs
`echo B > /world/knowledge/same.md`.

---

## B. Partial-failure recovery (9-16)

### B9. `spwn up` fails after chown, before probe
**Hypothesis:** container removed; state.json not polluted; next
`spwn up` is clean.
**Probe:** inject a fake tool whose verify always fails; observe
recovery.

### B10. docker cp fails mid-stream
**Hypothesis:** Architect.Spawn returns error with context; cleanup.
**Probe:** docker daemon stop mid-cp (hard to script; tree-test
fallback: very-long path).

### B11. Image build fails (no network during curl install)
**Hypothesis:** clean error pointing at which command failed, which
tool.
**Probe:** block claude.ai with /etc/hosts 127.0.0.1 in a fresh
container.

### B12. Disk full during SyncOut
**Hypothesis:** warning emitted, destroy still completes, leftover
partial write cleaned up.

### B13. Container crashed before Destroy called
**Hypothesis:** `spwn down` detects missing container, cleans state,
no panic.
**Probe:** `spwn up`, `docker kill <id>`, `spwn down`.

### B14. Agent mid-write when container is destroyed
**Hypothesis:** partial write in journal/ synced out; next spawn
sees it; no corruption.

### B15. `spwn up` with half-deleted image
**Hypothesis:** re-builds the missing layer cleanly.
**Probe:** `docker rmi` mid-run.

### B16. Ctrl-C during spawn
**Hypothesis:** context cancels, no zombie container, partial state
rolled back.

---

## C. State-machine holes (17-24)

### C17. `spwn down` on already-destroyed world
**Hypothesis:** exits 0 (idempotent) or clean "already destroyed"
message.

### C18. `spwn up` after `docker rm <container>` (manual)
**Hypothesis:** state.json has stale entry; `spwn ls` flags it;
re-running `up` either resurrects or re-creates.

### C19. Host reboot → `spwn ls` shows stale worlds
**Hypothesis:** containers gone, state says running; hydrate-from-
labels detects + marks stopped.

### C20. `spwn agent talk` on stopped container
**Hypothesis:** clean error "world not running, run spwn up first".

### C21. Orphan container (world-xxx exists but spwn doesn't know)
**Hypothesis:** `spwn ls` shows orphan or ignores silently.

### C22. Label mismatch (container has v1 labels, state.json v2)
**Hypothesis:** either migrate silently or surface a clear diagnose.

### C23. Two agents with same name in different worlds
**Hypothesis:** each has its own `~/.spwn/agents/<name>/worlds/<id>/`;
journals scoped by world-id; no cross-contamination.

### C24. Re-import an agent tar.gz over an existing agent
**Hypothesis:** refuses without --force OR prompts.

---

## D. Filesystem edges (25-34)

### D25. Agent name with spaces
**Probe:** `spwn agent new "my agent"`.
**Hypothesis:** either refused with format rule, or slugified.

### D26. Agent name with slashes / null bytes
**Probe:** `spwn agent new "foo/bar"`, `spwn agent new $'a\x00b'`.
**Hypothesis:** rejected at validation.

### D27. Agent name equal to reserved ("architect", "system")
**Hypothesis:** rejected OR creates shadowing issue to document.

### D28. Very long agent name (>200 chars)
**Hypothesis:** rejected or truncated deterministically.

### D29. Unicode agent name (日本, emoji)
**Hypothesis:** accepted if slug-safe, else clean error.

### D30. Case collision on macOS (Neo vs neo)
**Hypothesis:** project-level deduplication catches it.

### D31. Symlink inside workspace (escape attempt)
**Hypothesis:** docker's bind follows symlinks; agent sees symlink
target. Document as expected.

### D32. Binary file in `playbooks/`
**Probe:** `cp image.png spwn/agents/neo/playbooks/`, `spwn up`,
`spwn down`, check sync.
**Hypothesis:** SyncOut tar preserves it.

### D33. Very large playbook (10MB)
**Hypothesis:** handled without memory blow-up; renderer doesn't
crash on giant input.

### D34. Workspace path with spaces / special chars
**Probe:** `spwn up -w "/tmp/a b"`.
**Hypothesis:** bind-mount quoted correctly; works.

---

## E. Tool install failures (35-42)

### E35. Tool with non-zero install exit
**Probe:** local tool with `commands: [- "exit 1"]`.
**Hypothesis:** image build fails with "command returned exit 1",
pointing at the tool name.

### E36. Tool verify passes on an empty file (→ fixed in 3338f92c)
**Status:** already tracked; verify is weak but not directly
breakable today.

### E37. Two tools installing same binary
**Probe:** local tool A + B, both `cat > /usr/local/bin/collider`.
**Hypothesis:** second wins silently (OK) or errors.

### E38. Tool with infinite-loop install
**Hypothesis:** Docker build eventually times out; error surface.
**Probe:** local tool with `commands: [- "sleep 999999"]` +
--timeout.

### E39. Tool install that needs apt but apt cache was cleared
**Hypothesis:** clean error with "apt-get update required" hint.

### E40. Tool with circular dep (A deps B deps A)
**Probe:** author two local tools with mutual deps.
**Hypothesis:** resolver detects cycle, reports with both names.

### E41. Tool with missing `name:` in tool.yaml
**Hypothesis:** clean parse error with line number.

### E42. Tool whose verify command outputs warnings to stderr
**Hypothesis:** warnings don't fail the probe.

---

## F. Skill & playbook conflicts (43-50)

### F43. Two tools shipping a skill with the same name
**Probe:** install two deps that each ship `SKILL.md` at
`skills/<same-name>/SKILL.md`.
**Hypothesis:** second overwrites first (documented) or errors.

### F44. Skill name collides with Claude Code built-in
**Probe:** local `skill:review` (Claude Code has a built-in
`review`).
**Hypothesis:** spwn's version takes precedence via the
symlinked skills dir, or Claude Code prefers its own — either way
document.

### F45. Skill with 10 levels of `@-imports`
**Hypothesis:** Claude Code's 5-hop limit kicks in; graceful stop.

### F46. Skill with circular `@-import` (A→B→A)
**Hypothesis:** cycle-break at first re-visit.

### F47. Playbook filename = `secret-sauce.md` with BOM prefix
**Hypothesis:** parsed correctly — frontmatter still detected.

### F48. Playbook frontmatter with embedded newlines in description
**Probe:** `description: |\n  multi\n  line`.
**Hypothesis:** parser handles block scalar; first line used as
description (or whole body).

### F49. 100+ playbooks in one agent
**Hypothesis:** all indexed without crash; CLAUDE.md doesn't blow
up in size.

### F50. Playbook with unicode filename
**Hypothesis:** works; indexed + readable.

---

## G. Hook misbehavior (51-56)

### G51. Pre-spawn hook exits non-zero
**Hypothesis:** spawn aborts; no container created; error surfaces
with hook's stderr. ← **verified in the fix commit 73212cfa**.

### G52. Pre-spawn hook hangs forever
**Hypothesis:** spwn hangs? Should have a timeout; doesn't today.
**Probe:** `hook: sleep 999`; observe spawn.

### G53. Hook that `rm -rf /`-like destructive
**Hypothesis:** runs as-is (no sandboxing by design); document.

### G54. Hook that writes to spwn.yaml mid-flight
**Hypothesis:** the already-loaded Manifest isn't affected (in-
memory); next invocation sees the change.

### G55. Hook that prints binary to stderr
**Hypothesis:** stepper output may be garbled but no crash.

### G56. Two hooks with the same name (collision)
**Hypothesis:** only one script on disk; the install-time check
should reject second.

---

## H. Auth (57-60)

### H57. Missing anthropic credentials at spawn
**Probe:** move `~/.claude/.credentials.json` aside; `spwn up`.
**Hypothesis:** spawn succeeds; agent fails at talk time (no auth
token). Error clear.

### H58. Expired OAuth token mid-session
**Hypothesis:** claude auto-refreshes OR returns clear 401.

### H59. API key provider mismatch (openai token claimed as anthropic)
**Hypothesis:** spwn's auth layer surfaces a provider mismatch.

### H60. Corrupted `.credentials.json` (invalid JSON)
**Hypothesis:** PrelaunchShell copy succeeds but claude rejects;
talk errors clear.

---

## I. CLI UX (61-66)

### I61. `spwn agent talk` piped input (no TTY)
**Probe:** `echo "hi" | spwn agent talk neo` — sends?
**Hypothesis:** documented behavior.

### I62. SIGINT during `spwn agent talk`
**Hypothesis:** clean exit; session state preserved for next resume.

### I63. `spwn ls` with 100 worlds
**Hypothesis:** table paginates or renders full; no truncation crash.

### I64. JSON output with embedded newlines in description
**Hypothesis:** valid JSON (escaped).

### I65. CLI help output under 80 cols
**Hypothesis:** readable.

### I66. `spwn --json` on every command that should support it
**Probe:** survey of JSON support across `ls / inspect / agent ls /
world ls / check / auth / status`.

---

## J. Resource & snapshot (67-72)

### J67. PidsLimit=256 exceeded
**Probe:** agent spawns 300 processes; container should OOM or
kernel-block.

### J68. Snapshot of running container
**Probe:** `spwn up`, `spwn world snap save <id> s1`, verify image
tagged.

### J69. Restore snapshot → new world
**Probe:** `spwn world snap restore <snap-id>`; new world comes
online with preserved state.

### J70. Restore snapshot with project gone
**Hypothesis:** snapshot restore works even if project dir moved;
new world has stale role.md but otherwise fine.

### J71. Many snapshots (100+) — GC / cleanup
**Hypothesis:** no auto-GC today; document.

### J72. Delete snapshot that's in use by a running world
**Hypothesis:** `docker rmi` fails with "in use" error; clean.

---

## Total: 72 edge-case scenarios

## Execution plan

1. Script-runnable (majority): pipe into a bash harness similar to
   `../cli-scenarios/harness.sh`.
2. Docker-reliant: separate group; run serially to avoid daemon
   contention.
3. Agent-talk-reliant (few): run only when auth is live, batch them.

## Out of scope (still)

- DoS / adversarial workloads (testing against a malicious agent).
- Cross-platform (Windows/Linux divergence — spwn is macOS-first
  today).
- HTTP API auth edges (apps/api is same-machine only).
- Web UI specifics (Playwright suite).
