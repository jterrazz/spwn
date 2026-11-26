# Manual pass: CLI scenarios

Full path-based QA: each scenario is a **succession of real commands** a user
would run in a day, from a new project to teardown. Every scenario asserts at least one
outcome (exit code, file existence, substring in output).

Fifty scenarios, ninety-eight assertions. How a pass is run is [Run a manual pass](../guides/run-a-manual-pass.md).

## How to rerun

```bash
make build && make test-image
bash scripts/manual-harness.sh            # all 50
bash scripts/manual-harness.sh 31 40      # the Docker ones
```

`SPWN` names another binary, `TMP_ROOT` another scratch root (default `/tmp/qa-50`); `SPWN_BASE_IMAGE` defaults to the mock image `spwn-test:latest`. The harness exits non-zero with a failure count.

## Isolation

- All user-level state lives under `$SPWN_HOME` (default `/tmp/qa-50/spwn_home`).
  The developer's real `~/.spwn` is never touched.
- Each scenario gets its own `$TMP_ROOT/sNN/` dir and `cd`s into it, then into the project it creates there (`./p`, or the example's name), so project
  state doesn't leak across scenarios.
- Every scenario starts by `docker rm -f` on any leftover `world-*` containers
  from a prior run.

## Out of scope (intentionally not in this pass)

- `spwn auth login` (interactive OAuth flow)

---

## Scenario catalog

### Group A: First-run lifecycle (1 to 10)

#### 1. Fresh user journey: create, plan, JSON validation

```bash
cd $TMP/s01
station project create p && cd p             # → scaffold created in ./p
station project plan                                   # → exit 0, "Project is valid"
station project plan --json                            # → {"valid": true, "summary": {...}}
```

**Asserts:** `create` exit 0 · `plan` exit 0 · `plan --json` contains `"valid": true`.

#### 2. `station project create` never creates over a folder

```bash
cd $TMP/s02
station project create p                     # → exit 0
station project create p                     # → exit != 0 (the folder exists)
cd p && station project plan                 # → exit 0
```

**Asserts:** first `create` zero · second `create` nonzero · `plan` zero.

#### 3. `station project create --from matrix`

```bash
cd $TMP/s03
station project create --from matrix && cd matrix  # the catalog's example
ls spwn/agents/neo                           # → neo is present
station project plan                                   # → exit 0
```

**Asserts:** create zero · `spwn/agents/neo/` exists · plan zero.

#### 4. `station project create --from startup`, multi-agent

```bash
cd $TMP/s04
station project create --from startup && cd startup
station project plan
station agents --json                         # → includes ceo, devops, analyst
```

**Asserts:** create zero · plan zero · `ceo`/`devops`/`analyst` each in JSON.

#### 5. An example the catalog does not hold

```bash
station project create --from does-not-exist-xyz-987  # → exit != 0
```

**Asserts:** nonzero exit.

#### 6. `spwn up` outside any spwn.yaml (global-mode fallback)

Global mode is the legacy fallback. Should **not crash**: either falls back
to the default world config, `~/.spwn/worlds/default.yaml` or the built-in
one when that file is absent (rc 0), or errors cleanly (rc 1).

```bash
cd $TMP/s06                                  # no spwn.yaml here
spwn up                                      # → rc ∈ {0,1}, never a panic
```

**Asserts:** rc ≤ 1.

#### 7. Double `spwn up` (idempotency)

```bash
cd $TMP/s07
station project create --from matrix && cd matrix
spwn up                                      # → exit 0
spwn up                                      # → exit 0 or 1, never a panic
spwn down
```

**Asserts:** first `up` zero · second `up` rc ≤ 1.

_Note:_ uses `spwn:matrix` scaffold because the default scaffold ships with
`tool:greet` + `skill:focus` + `hook:pre-spawn` local refs that don't exist
in a prebuilt `SPWN_BASE_IMAGE`. Matrix has only `spwn:*` deps.

#### 8. `spwn down` with no worlds

```bash
cd $TMP/s08
station project create p && cd p
spwn down                                    # → rc ≤ 1, never a panic
```

**Asserts:** rc ≤ 1.

#### 9. Malformed `spwn.yaml`: `station project plan` errors precisely

```bash
printf "invalid: [yaml broken" > spwn.yaml
station project plan                                   # → exit != 0
```

**Asserts:** nonzero exit.

#### 10. Invalid dep ref in agent.yaml: check catches it

```bash
station project create p && cd p
# flip "spwn:unix" to "spwn:does-not-exist-zz" in spwn/agents/neo/agent.yaml
station project plan                                   # → exit != 0
```

**Asserts:** nonzero exit.

---

### Group B: Agent CRUD (11 to 20)

#### 11. `station agents create` scaffolds the home tree

```bash
station project create p && cd p && station agents create bob
ls spwn/agents/bob                           # → agent.yaml, SOUL.md, AGENTS.md, playbooks/, journal/
```

**Asserts:** `new` zero · `agent.yaml` + `SOUL.md` both present.

#### 12. Duplicate `station agents create` refuses

```bash
station project create p && cd p && station agents create bob
station agents create bob                           # → exit != 0
```

#### 13. `station agents` in plain + JSON

```bash
station project create p && cd p && station agents create bob
station agents
station agents --json                         # → valid JSON, contains bob + neo
```

#### 14. `station agents <name>` happy + missing

```bash
station project create p && cd p
station agents neo                       # → exit 0
station agents does-not-exist            # → exit != 0
```

#### 15. `station agents delete` happy + missing

```bash
station project create p && cd p && station agents create bob
station agents delete bob --confirm=bob                            # → exit 0
station agents delete bob --confirm=bob                            # → exit != 0
```

#### 16. `station agents export` produces a gzip archive

```bash
station project create p && cd p && station agents create bob
station agents export bob --to .                        # → bob*.tar.gz on disk
file bob*.tar.gz                             # → "gzip compressed data"
```

#### 17. `station agents import` round-trip

```bash
station project create p && cd p && station agents create bob && station agents export bob --to .
station agents delete bob --confirm=bob
station agents import bob*.tar.gz                # → exit 0
station agents --json                         # → contains bob
```

#### 18. `station agents fork` clones identity + memory

```bash
station project create p && cd p && station agents create bob
station agents fork bob --as bob-v2                   # → exit 0
station agents --json                         # → contains bob and bob-v2
```

#### 19. `station agents <name>`

```bash
station project create p && cd p && station agents create bob
station agents bob                       # → just bob, exit 0
station agents does-not-exist            # → exit != 0
```

#### 20. `station skills create / list / delete`

```bash
station project create p && cd p
station skills create daily-standup          # → writes spwn/skills/daily-standup.md
station skills                               # → exit 0
station skills delete daily-standup --confirm=daily-standup  # → exit 0; file gone
```

---

### Group C: Dependencies + lockfile (21 to 30)

#### 21. `spwn install python`, global install

```bash
station project create p && cd p && spwn install python
grep python spwn/agents/neo/agent.yaml       # → present
grep python spwn.lock                        # → present
```

#### 22. `spwn install node --agent bob`, scoped install

```bash
station project create p && cd p && station agents create bob
spwn install node --agent bob
# → bob's agent.yaml has spwn:node; neo's does NOT
```

#### 23. `spwn uninstall python`

```bash
station project create p && cd p && spwn install python
spwn uninstall python                        # → exit 0; python gone
```

#### 24. Bogus install ref does not pollute the lockfile

```bash
spwn install spwn:does-not-exist-xyz         # → exit != 0
grep does-not-exist-xyz spwn.lock            # → not present
```

#### 25. **[Bug fix regression guard]** `spwn install skill:nonexistent --agent Y` is refused

Before the fix: silently succeeded, added a broken ref. Now: fails fast with
`skill:<name> not found at spwn/skills/<name>.md; create it first with
\`station skills create <name>\``.

```bash
station project create p && cd p
spwn install skill:does-not-exist --agent neo
# → exit != 0 (was 0 pre-fix)
```

#### 26. Bare name auto-promotes to `spwn:<name>`

```bash
station project create p && cd p && spwn install node               # → becomes spwn:node
grep "spwn:node\|node" spwn/agents/neo/agent.yaml   # → present
```

#### 27. `spwn install github:...` (planned feature path)

```bash
spwn install github:jterrazz/does-not-exist  # → rc ≤ 1 (clean error OR planned-stub exit)
```

#### 28. Local refs refused without `--agent`

```bash
station project create p && cd p && echo "# test" > spwn/skills/myskill.md
spwn install skill:myskill                   # → exit != 0 (no --agent)
```

#### 29. Lockfile deterministic ordering

```bash
station project create p && cd p && spwn install python && spwn install node
diff <(grep '^spwn:' spwn.lock) <(grep '^spwn:' spwn.lock | sort)
# → zero diff
```

#### 30. Repeated `spwn install` keeps project valid

```bash
station project create p && cd p
spwn install python
spwn install node
spwn install git
station project plan                                   # → exit 0
```

---

### Group D: Real Docker, spawn + destroy + mounts (31 to 40)

These need `SPWN_BASE_IMAGE=spwn-test:latest`, the mock-claude test image built
by `make test-image`. Without it, each `spwn up` burns ~5 minutes building
a real production image.

#### 31. matrix scaffold spawns; `spwn world list` shows running

```bash
station project create --from matrix && cd matrix && spwn up
spwn world list                              # → table contains "running"
spwn down
```

#### 32. **[Hour-2 regression guard]** `spwn world inspect` shows project-rooted home

Pre-fix: `Agent home: ~/.spwn/agents/neo → /agents/neo`. Post-fix: the project's
`spwn/agents/neo/` path. Scenario regex-matches `/tmp/qa-50/sNN/spwn/agents/` in
the inspect output.

```bash
station project create --from matrix && cd matrix && spwn up
spwn world inspect $(docker ps --format '{{.Names}}' --filter name=world-)
# → output includes ".../spwn/agents/<name>" path
```

#### 33. `spwn world list --json` is valid JSON

```bash
station project create --from matrix && cd matrix && spwn up
spwn world list --json                       # → parseable JSON, has "worlds" key
spwn down
```

#### 34. `spwn down <id>` destroys a single world

```bash
station project create --from matrix && cd matrix && spwn up
spwn down world-matrix-<runtime-suffix>      # → exit 0
```

#### 35. `spwn world list` shows nothing after `spwn down`

```bash
station project create --from matrix && cd matrix && spwn up && spwn down
spwn world list --json 2>&1                # → no "running"
```

#### 36. `spwn logs --world` (config name) + `spwn world logs <id>` (runtime id)

Two forms:

```bash
station project create --from matrix && cd matrix && spwn up
spwn logs --world matrix                     # → exit 0 (config name)
spwn world logs world-matrix-<suffix>        # → exit 0 (runtime id)
spwn down
```

#### 37. `spwn world inspect` happy + not-found

```bash
station project create --from matrix && cd matrix && spwn up
spwn world inspect <real-id>                 # → exit 0
spwn world inspect world-does-not-exist-12345  # → exit != 0
```

#### 38. Knowledge mount propagates into `/world/knowledge/`

```bash
station project create --from matrix && cd matrix
mkdir knowledge && echo "test fact" > knowledge/fact.md
# add "knowledge: ./knowledge" to the matrix entry in spwn.yaml
spwn up
docker exec <id> cat /world/knowledge/fact.md   # → "test fact"
spwn down
```

#### 39. Workspaces mount at `/workspaces/<name>/`

```bash
station project create --from matrix && cd matrix
spwn up -w .                                 # mount cwd as a workspace
docker exec <id> ls /workspaces/             # → non-empty
```

#### 40. After `spwn down`, matrix containers are gone

```bash
station project create --from matrix && cd matrix && spwn up
# docker ps filter name=world-matrix → one container
spwn down
# docker ps filter name=world-matrix → empty
```

---

### Group E: Errors + edge cases (41 to 50)

#### 41. Malformed YAML: check gives helpful error

```bash
printf "version: 1\nname: bad\nworlds:\n  mat: [ broken\n" > spwn.yaml
station project plan                                   # → exit != 0, mentions "yaml" or "line"
```

#### 42. Unicode agent name is handled cleanly

```bash
station project create p && cd p && station agents create 日本              # rc ≤ 1 (reject or slugify, no crash)
```

#### 43. Very long agent name (>63 chars, slug boundary)

```bash
station project create p && cd p && station agents create $(printf 'x%.0s' {1..80})   # rc ≤ 1
```

#### 44. Empty `dependencies: []` is valid

```bash
station project create p && cd p
# rewrite spwn/agents/neo/agent.yaml with dependencies: []
station project plan                                   # → exit 0
```

#### 45. Corrupted `spwn.lock` recovers via install

```bash
station project create p && cd p
printf "garbage content\n" > spwn.lock
station project plan                                   # → rc ≤ 1 (either flags or tolerates)
spwn install python                          # → exit 0 (regenerates lock)
```

#### 46. Project discovery walks upward

```bash
station project create p && cd p && mkdir -p a/b/c && cd a/b/c
station project plan                                   # → finds spwn.yaml two dirs up, exit 0
```

#### 47. Sibling projects are isolated

```bash
station project create p1 && (cd p1 && station agents create alice)
station project create p2 && (cd p2 && station agents create bob)
cd p1 && station agents --json               # → alice present, bob absent
```

#### 48. `spwn --version`

```bash
spwn --version                               # → exit 0
```

#### 49. Top-level help banner

```bash
spwn --help                                  # → contains "Quick Start" + "Entities"
spwn help                                    # → same (alias)
```

#### 50. Full three-verb sanity: create, up, exec-inside, down

```bash
station project create --from matrix && cd matrix                        # → exit 0
station project plan                                   # → exit 0
spwn up                                      # → exit 0
docker exec <id> ls /agents/neo/CLAUDE.md    # → exit 0 (file exists inside)
spwn down                                    # → exit 0
```

---

## Harness notes

- The harness is plain bash with Python one-liners: `run` asserts an exit code, `checkstr` a substring, and a scenario adds its own checks inline.
- The Docker group (31 to 40) takes about two minutes, the other forty about thirty seconds, with `SPWN_BASE_IMAGE` set.
- A clean run ends on `RESULTS: 98 passed, 0 failed`; any failure is a regression against a scenario above.
