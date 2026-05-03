# spwn CLI — 50-Scenario QA Pass

Full path-based QA: each scenario is a **succession of real commands** a user
would run in a day, from init to teardown. Every scenario asserts at least one
outcome (exit code, file existence, substring in output).

- **Total scenarios:** 50
- **Total sub-assertions:** 98
- **Last run:** all green against commit `da978c1c` on `main`
- **Real bugs found + fixed on this pass:** 2 (see "Bugs surfaced" at bottom)

## How to rerun

```bash
# Prereqs
make -C /path/to/spwn/spwn build               # produces bin/spwn
make -C /path/to/spwn/spwn build-test-image    # produces spwn-test:latest

# Point harness at the binary (defaults to the path used on this machine)
SPWN=/path/to/spwn/spwn/bin/spwn \
  bash ./harness.sh            # run all 50
bash ./harness.sh 1 15         # subset
bash ./harness.sh 31 40        # Docker ones

# Env knobs
SPWN_HOME=/tmp/qa-50/spwn_home        # isolated user-level state (auto-set)
SPWN_BASE_IMAGE=spwn-test:latest      # mock image so `spwn up` doesn't burn 5min
TMP_ROOT=/tmp/qa-50                   # scenario scratch dirs
```

The harness exits non-zero with a failure count if anything regresses.

## Isolation

- All user-level state lives under `$SPWN_HOME` (default `/tmp/qa-50/spwn_home`).
  The developer's real `~/.spwn` is never touched.
- Each scenario gets its own `$TMP_ROOT/sNN/` dir and `cd`s into it so project
  state doesn't leak across scenarios.
- Every scenario starts by `docker rm -f` on any leftover `world-*` containers
  from a prior run.

## Out of scope (intentionally not in this pass)

- `spwn architect start` (DooD setup is heavy; exercised by world E2E tests instead)
- `spwn auth login` (interactive OAuth flow)
- `spwn web` (browser launch)
- Playwright web UI specs (separate bucket: `make test-web`)

---

# Scenario catalog

## Group A — First-run lifecycle (1–10)

### 1. Fresh user journey — init → check → JSON validation

```bash
cd $TMP/s01
spwn init                                    # → scaffold created
spwn check                                   # → exit 0, "Project is valid"
spwn check --json                            # → {"valid": true, "summary": {...}}
```

**Asserts:** `init` exit 0 · `check` exit 0 · `check --json` contains `"valid": true`.

### 2. `spwn init` refuses to clobber

```bash
cd $TMP/s02
echo "version: 1" > spwn.yaml                # dirty dir
spwn init                                    # → exit != 0
spwn init --force                            # → exit 0 (overwrites)
spwn check                                   # → exit 0
```

**Asserts:** first `init` nonzero · `init --force` zero · `check` zero.

### 3. `spwn init spwn:matrix`

```bash
cd $TMP/s03
spwn init spwn:matrix                        # catalog scaffold
ls spwn/agents/neo                           # → neo is present
spwn check                                   # → exit 0
```

**Asserts:** init zero · `spwn/agents/neo/` exists · check zero.

### 4. `spwn init spwn:startup` — multi-agent

```bash
cd $TMP/s04
spwn init spwn:startup
spwn check
spwn agent ls --json                         # → includes ceo, devops, analyst
```

**Asserts:** init zero · check zero · `ceo`/`devops`/`analyst` each in JSON.

### 5. Bogus init ref

```bash
spwn init spwn:does-not-exist-xyz-987        # → exit != 0
```

**Asserts:** nonzero exit.

### 6. `spwn up` outside any spwn.yaml (global-mode fallback)

Global mode is the legacy fallback. Should **not crash** — either falls back
to `~/.spwn/worlds/default.yaml` (rc 0) or errors cleanly (rc 1).

```bash
cd $TMP/s06                                  # no spwn.yaml here
spwn up                                      # → rc ∈ {0,1}, never a panic
```

**Asserts:** rc ≤ 1.

### 7. Double `spwn up` (idempotency)

```bash
cd $TMP/s07
spwn init spwn:matrix
spwn up                                      # → exit 0
spwn up                                      # → exit 0 or 1, never a panic
spwn down
```

**Asserts:** first `up` zero · second `up` rc ≤ 1.

_Note:_ uses `spwn:matrix` scaffold because the default scaffold ships with
`tool:greet` + `skill:focus` + `hook:pre-spawn` local refs that don't exist
in a prebuilt `SPWN_BASE_IMAGE`. Matrix has only `spwn:*` deps.

### 8. `spwn down` with no worlds

```bash
cd $TMP/s08
spwn init
spwn down                                    # → rc ≤ 1, never a panic
```

**Asserts:** rc ≤ 1.

### 9. Malformed `spwn.yaml` → `spwn check` errors precisely

```bash
printf "invalid: [yaml broken" > spwn.yaml
spwn check                                   # → exit != 0
```

**Asserts:** nonzero exit.

### 10. Invalid dep ref in agent.yaml → check catches it

```bash
spwn init
# flip "spwn:unix" to "spwn:does-not-exist-zz" in spwn/agents/neo/agent.yaml
spwn check                                   # → exit != 0
```

**Asserts:** nonzero exit.

---

## Group B — Agent CRUD (11–20)

### 11. `spwn agent new` scaffolds the home tree

```bash
spwn init && spwn agent new bob
ls spwn/agents/bob                           # → agent.yaml, SOUL.md, AGENTS.md, playbooks/, journal/
```

**Asserts:** `new` zero · `agent.yaml` + `SOUL.md` both present.

### 12. Duplicate `spwn agent new` refuses without `--force`

```bash
spwn init && spwn agent new bob
spwn agent new bob                           # → exit != 0
spwn agent new bob --force                   # → exit 0
```

### 13. `spwn agent ls` in plain + JSON

```bash
spwn init && spwn agent new bob
spwn agent ls
spwn agent ls --json                         # → valid JSON, contains bob + neo
```

### 14. `spwn agent inspect` happy + missing

```bash
spwn init
spwn agent inspect neo                       # → exit 0
spwn agent inspect does-not-exist            # → exit != 0
```

### 15. `spwn agent rm` happy + missing

```bash
spwn init && spwn agent new bob
spwn agent rm bob                            # → exit 0
spwn agent rm bob                            # → exit != 0
```

### 16. `spwn agent export` produces a gzip archive

```bash
spwn init && spwn agent new bob
spwn agent export bob                        # → bob*.tar.gz on disk
file bob*.tar.gz                             # → "gzip compressed data"
```

### 17. `spwn agent import` round-trip

```bash
spwn init && spwn agent new bob && spwn agent export bob
spwn agent rm bob
spwn agent import bob*.tar.gz                # → exit 0
spwn agent ls --json                         # → contains bob
```

### 18. `spwn agent fork` clones identity + memory

```bash
spwn init && spwn agent new bob
spwn agent fork bob bob-v2                   # → exit 0
spwn agent ls --json                         # → contains bob and bob-v2
```

### 19. `spwn inspect` (project-wide and per-agent)

```bash
spwn init && spwn agent new bob
spwn inspect                                 # → both agents
spwn inspect bob                             # → just bob, exit 0
spwn inspect does-not-exist                  # → exit != 0
```

### 20. `spwn skill new / ls / rm`

```bash
spwn init
spwn skill new daily-standup                 # → writes spwn/skills/daily-standup.md
spwn skill ls                                # → exit 0
spwn skill rm daily-standup                  # → exit 0; file gone
```

---

## Group C — Dependencies + lockfile (21–30)

### 21. `spwn install python` — global install

```bash
spwn init && spwn install python
grep python spwn/agents/neo/agent.yaml       # → present
grep python spwn.lock                        # → present
```

### 22. `spwn install node --agent bob` — scoped install

```bash
spwn init && spwn agent new bob
spwn install node --agent bob
# → bob's agent.yaml has spwn:node; neo's does NOT
```

### 23. `spwn uninstall python`

```bash
spwn init && spwn install python
spwn uninstall python                        # → exit 0; python gone
```

### 24. Bogus install ref does not pollute the lockfile

```bash
spwn install spwn:does-not-exist-xyz         # → exit != 0
grep does-not-exist-xyz spwn.lock            # → not present
```

### 25. **[Bug fix regression guard]** `spwn install skill:nonexistent --agent Y` is refused

Before the fix: silently succeeded, added a broken ref. Now: fails fast with
`skill:<name> not found at spwn/skills/<name>.md — create it first with
\`spwn skill new <name>\``.

```bash
spwn init
spwn install skill:does-not-exist --agent neo
# → exit != 0 (was 0 pre-fix)
```

### 26. Bare name auto-promotes to `spwn:<name>`

```bash
spwn init && spwn install node               # → becomes spwn:node
grep "spwn:node\|node" spwn/agents/neo/agent.yaml   # → present
```

### 27. `spwn install github:...` (planned feature path)

```bash
spwn install github:jterrazz/does-not-exist  # → rc ≤ 1 (clean error OR planned-stub exit)
```

### 28. Local refs refused without `--agent`

```bash
spwn init && echo "# test" > spwn/skills/myskill.md
spwn install skill:myskill                   # → exit != 0 (no --agent)
```

### 29. Lockfile deterministic ordering

```bash
spwn init && spwn install python && spwn install node
diff <(grep '^spwn:' spwn.lock) <(grep '^spwn:' spwn.lock | sort)
# → zero diff
```

### 30. Repeated `spwn install` keeps project valid

```bash
spwn init
spwn install python
spwn install node
spwn install git
spwn check                                   # → exit 0
```

---

## Group D — Real Docker: spawn + destroy + mounts (31–40)

These need `SPWN_BASE_IMAGE=spwn-test:latest` — the mock-claude test image built
by `make build-test-image`. Without it, each `spwn up` burns ~5 minutes building
a real production image.

### 31. matrix scaffold spawns; `spwn ls` shows running

```bash
spwn init spwn:matrix && spwn up
spwn ls                                      # → table contains "running"
spwn down
```

### 32. **[Hour-2 regression guard]** `spwn world inspect` shows project-rooted home

Pre-fix: `Agent home: ~/.spwn/agents/neo → /agents/neo`. Post-fix: the project's
`spwn/agents/neo/` path. Scenario regex-matches `/tmp/qa-50/sNN/spwn/agents/` in
the inspect output.

```bash
spwn init spwn:matrix && spwn up
spwn world inspect $(docker ps --format '{{.Names}}' --filter name=world-)
# → output includes ".../spwn/agents/<name>" path
```

### 33. `spwn world ls --json` is valid JSON

```bash
spwn init spwn:matrix && spwn up
spwn world ls --json                         # → parseable JSON, has "worlds" key
spwn down
```

### 34. `spwn down <id>` destroys a single world

```bash
spwn init spwn:matrix && spwn up
spwn down world-matrix-<runtime-suffix>      # → exit 0
```

### 35. `spwn ls` shows nothing after `spwn down`

```bash
spwn init spwn:matrix && spwn up && spwn down
spwn ls --json 2>&1                          # → no "running"
```

### 36. `spwn logs --world` (config name) + `spwn world logs <id>` (runtime id)

Two forms:

```bash
spwn init spwn:matrix && spwn up
spwn logs --world matrix                     # → exit 0 (config name)
spwn world logs world-matrix-<suffix>        # → exit 0 (runtime id)
spwn down
```

### 37. `spwn world inspect` happy + not-found

```bash
spwn init spwn:matrix && spwn up
spwn world inspect <real-id>                 # → exit 0
spwn world inspect world-does-not-exist-12345  # → exit != 0
```

### 38. Knowledge mount propagates into `/world/knowledge/`

```bash
spwn init spwn:matrix
mkdir knowledge && echo "test fact" > knowledge/fact.md
# add "knowledge: ./knowledge" to the matrix entry in spwn.yaml
spwn up
docker exec <id> cat /world/knowledge/fact.md   # → "test fact"
spwn down
```

### 39. Workspaces mount at `/workspaces/<name>/`

```bash
spwn init spwn:matrix
spwn up -w .                                 # mount cwd as a workspace
docker exec <id> ls /workspaces/             # → non-empty
```

### 40. After `spwn down`, matrix containers are gone

```bash
spwn init spwn:matrix && spwn up
# docker ps filter name=world-matrix → one container
spwn down
# docker ps filter name=world-matrix → empty
```

---

## Group E — Errors + edge cases (41–50)

### 41. Malformed YAML → check gives helpful error

```bash
printf "version: 1\nname: bad\nworlds:\n  mat: [ broken\n" > spwn.yaml
spwn check                                   # → exit != 0, mentions "yaml" or "line"
```

### 42. Unicode agent name is handled cleanly

```bash
spwn init && spwn agent new 日本              # rc ≤ 1 (reject or slugify, no crash)
```

### 43. Very long agent name (>63 chars — slug boundary)

```bash
spwn init && spwn agent new $(printf 'x%.0s' {1..80})   # rc ≤ 1
```

### 44. Empty `dependencies: []` is valid

```bash
spwn init
# rewrite spwn/agents/neo/agent.yaml with dependencies: []
spwn check                                   # → exit 0
```

### 45. Corrupted `spwn.lock` recovers via install

```bash
spwn init
printf "garbage content\n" > spwn.lock
spwn check                                   # → rc ≤ 1 (either flags or tolerates)
spwn install python                          # → exit 0 (regenerates lock)
```

### 46. Project discovery walks upward

```bash
spwn init && mkdir -p a/b/c && cd a/b/c
spwn check                                   # → finds spwn.yaml two dirs up, exit 0
```

### 47. Sibling projects are isolated

```bash
mkdir p1 p2
(cd p1 && spwn init && spwn agent new alice)
(cd p2 && spwn init && spwn agent new bob)
cd p1 && spwn agent ls --json               # → alice present, bob absent
```

### 48. `spwn --version`

```bash
spwn --version                               # → exit 0
```

### 49. Top-level help banner

```bash
spwn --help                                  # → contains "Quick Start" + "Entities"
spwn help                                    # → same (alias)
```

### 50. Full three-verb sanity — init → up → exec-inside → down

```bash
spwn init spwn:matrix                        # → exit 0
spwn check                                   # → exit 0
spwn up                                      # → exit 0
docker exec <id> ls /agents/neo/CLAUDE.md    # → exit 0 (file exists inside)
spwn down                                    # → exit 0
```

---

# Bugs surfaced + fixed on this pass

Both shipped in commit `da978c1c`.

## Bug 1 — `spwn install skill:nonexistent --agent Y` silently accepted

- **Scenario:** #25
- **Repro:**
  ```
  spwn init && spwn install skill:does-not-exist --agent neo
  ```
  Pre-fix: exit 0, `skill:does-not-exist` written to `agent.yaml` and
  `spwn.lock`. Failure only surfaces later at `spwn up` as a resolver error.
- **Root cause:** `RunInstall` validated the ref scheme but never confirmed
  the file existed on disk for local refs.
- **Fix:** `apps/cli/dependency/dependency.go` now calls `refs.ResolveSkill`
  for `KindLocalSkill`/`KindLocalTool`/`KindLocalHook` and returns a targeted
  error message naming the expected path.

## Bug 2 — `spwn logs --world` help text misled users

- **Scenario:** #36
- **Repro:**
  ```
  spwn up && spwn logs --world world-matrix-abc   # with a real runtime id
  ```
  Fails with "unknown world — not declared in spwn.yaml" because `--world`
  actually wants a config name from `spwn.yaml#worlds` (e.g. `matrix`). The
  per-id form is `spwn world logs <id>`.
- **Fix:** `apps/cli/logs/logs.go` flag description now says:
  > Filter by world config name (e.g. 'matrix' from spwn.yaml#worlds). Use
  > `spwn world logs <id>` to filter by a runtime world ID.

---

# Appendix — Harness notes

- 50 scenarios, 98 sub-assertions (some scenarios test multiple conditions).
- Runtime on this machine: ~2 min for the Docker group (31–40), ~30s for the
  other 40. Full run ≈ 3 min with `SPWN_BASE_IMAGE` set.
- Harness is plain bash + Python one-liners — no test framework. `run` asserts
  exit code; `checkstr` asserts a substring; scenarios can add ad-hoc checks
  inline. See `./harness.sh`.
- The harness mutates `$TMP_ROOT` (default `/tmp/qa-50/`) but never touches the
  host's real `~/.spwn` — `SPWN_HOME` points at `$TMP_ROOT/spwn_home`.

## Re-running after a pull

```bash
cd /path/to/spwn/spwn
git pull
make build                                   # rebuild bin/spwn
make build-test-image                        # rebuild spwn-test:latest
bash ./harness.sh
# Expect: "RESULTS: 98 passed, 0 failed"
```

Any new failure indicates a regression against one of the behaviors above.
