# spwn — 50 Deep Agent-Behavior QA Scenarios

> **Human-driven QA:** every scenario requires a real Anthropic-authenticated
> Claude session running inside a spwn world. The scenarios probe whether the
> host-side setup (CLAUDE.md, playbooks, skills, hooks, tools, knowledge,
> roster, messaging) is **correctly injected** into the agent's perception —
> i.e. the agent actually sees, reads, and acts on what spwn claims to have
> configured.
>
> These cannot be run by the script harness in `../cli-scenarios/harness.sh` (which
> uses mock-claude). A human runs each scenario, interacts with the live agent
> via `spwn agent talk`, and verifies the responses match the PASS criteria.

## Prerequisites

- `spwn auth` shows a connected Anthropic provider (OAuth or token).
- `make build` + `make build-test-image` completed once.
- A scratch project dir per scenario under `$TMP/agentqa/sNN/` (commit it to
  a disposable git repo if you want diffs).
- The agent runs claude-code natively; don't set `SPWN_BASE_IMAGE` unless a
  scenario explicitly says so — we need the real runtime.

## Conventions used in this document

- **Setup** — host-side config before spawn.
- **Spawn** — the `spwn up` line(s) used to boot the world.
- **Prompt sequence** — exact strings to paste into the agent chat (via
  `spwn agent talk <name>` interactive mode, or `spwn agent send` for async).
- **Expect** — agent behaviors / substrings that must be observable.
- **PASS criteria** — binary: either all Expect items hold or the scenario
  fails.
- **Likely failure modes** — what a broken injection tends to look like, so
  the tester can diagnose fast.

Keep a scorecard per scenario. A scenario is ✅ only when every PASS criterion
is satisfied; partial success is ❌ with notes.

---

# Group A — Identity & self-perception (1-10)

Goal: does the agent correctly perceive *who it is* per SOUL.md, its `CLAUDE.md`,
its role in the world, and the agent.yaml composition?

### Scenario 1 — Agent reads its SOUL.md first

**Setup:**
```bash
mkdir -p $TMP/agentqa/s01 && cd $TMP/agentqa/s01
spwn init
cat > spwn/agents/neo/SOUL.md <<'EOF'
# Neo — Oracle of /workspaces/

You are Neo. You speak in short, measured lines. You always prefix observations
with "I observe:". You refuse to speculate.
EOF
```

**Spawn:** `spwn up`

**Prompt sequence:**
1. `who are you?`
2. `what's your voice pattern?`
3. `what will you never do?`
4. `what file defines this identity? read it and quote the first line.`

**Expect:**
- Agent identifies as "Neo" in the first reply.
- Response style matches SOUL.md: short, starts with "I observe:".
- Answer to #3 references "speculate" / "refuse to speculate".
- #4: agent opens `/agents/neo/SOUL.md` (or `./SOUL.md` since cwd is the home)
  and quotes `# Neo — Oracle of /workspaces/`.

**PASS criteria:** all four observed.

**Likely failure modes:**
- Agent introduces itself as "Claude" instead of Neo → `@SOUL.md` import in
  CLAUDE.md didn't resolve.
- Voice pattern absent → SOUL.md exists but agent never read it.
- File-read attempt returns wrong path → CLAUDE.md's `@SOUL.md` relative-path
  convention misinterpreted.

---

### Scenario 2 — Agent knows its role in this specific world

**Setup:**
```bash
cd $TMP/agentqa/s02 && spwn init
# Edit spwn.yaml to name this world "production-audit":
sed -i '' 's/neo:/production-audit:/' spwn.yaml
```

**Spawn:** `spwn up`

**Prompts:**
1. `what is the name of the world you're deployed in?`
2. `quote the "Role here" section of your CLAUDE.md verbatim.`
3. `what agent name does the world container think you have?`

**Expect:**
- #1: agent says `production-audit` (or the runtime world-id starting with
  `world-production-audit-`).
- #2: agent quotes the `## Role here` block that the runtime renderer
  inlines into each agent's CLAUDE.md (e.g. `You are deployed as a worker
  in world-production-audit-<id>.`).
- #3: `neo`.

**PASS:** all three correct and the agent actually ran a `cat` / `read file`
operation on CLAUDE.md (observable in its tool use).

**Likely failure modes:**
- Agent guesses the world name from the container hostname rather than
  reading CLAUDE.md → the "Role here" inlining broke or the world-id
  isn't landing in the rendered block.

**Note (2026-04-22):** Earlier revisions of this scenario expected a
`worlds/<id>/role.md` file; that was removed in commit `d06f1517`
("inline role into CLAUDE.md/AGENTS.md; drop worlds/<id>/role.md") —
the role now lives as a `## Role here` heading inside CLAUDE.md /
AGENTS.md.

---

### Scenario 3 — Agent perceives its two-layer Mind (playbooks + journal only)

**Setup:**
```bash
cd $TMP/agentqa/s03 && spwn init
```

**Prompts:**
1. `list every directory under /agents/neo/ that you are supposed to own.`
2. `are "skills/" or "knowledge/" Mind layers?`
3. `where do your durable procedures live?`
4. `where are your session histories?`

**Expect:**
- #1: agent names `SOUL.md`, `playbooks/`, `journal/` (possibly also
  `agent.yaml`, `AGENTS.md`, `worlds/`). No skills/ or knowledge/ under the
  agent home.
- #2: agent says no — skills are build-time deps in `/world/skills/`,
  knowledge is world-scoped at `/world/knowledge/`.
- #3: `playbooks/`.
- #4: `journal/`.

**PASS:** agent correctly identifies the 2-layer Mind model and does NOT
claim skills or knowledge are per-agent layers.

---

### Scenario 4 — Agent describes its declared dependencies

**Setup:**
```bash
cd $TMP/agentqa/s04 && spwn init
spwn install python node
```

**Prompts:**
1. `list every dependency you have installed.`
2. `for each, confirm the binary exists by running which <binary>.`
3. `do you have Docker access?`

**Expect:**
- #1: `spwn:unix`, `spwn:git`, `spwn:python`, `spwn:node`, the claude-code
  runtime — plus scaffold locals if the default scaffold was used (`skill:focus`,
  `tool:greet`, `hook:pre-spawn`).
- #2: `which python`, `which node`, `which git`, `which greet` all return
  paths inside the container; docker isn't in the list so `which docker`
  returns non-zero.
- #3: no (agents don't have Docker socket by default).

---

### Scenario 5 — Agent knows its physics (Laws + Topology + Communication)

**Prompts:**
1. `describe the Laws governing your filesystem. be specific.`
2. `what paths are ephemeral, and what persists?`
3. `if you want to message another agent, what path do you write to?`

**Expect:**
- #1: "Network: bridge", "Filesystem is ephemeral except /workspaces and
  /agents".
- #2: `/tmp` is ephemeral, `/world/*` mostly ephemeral except knowledge,
  `/agents/<name>/` persists across worlds, `/workspaces/<name>/` is host-
  mounted.
- #3: `/world/inbox/<recipient>/<timestamp>-from-<sender>.md` (from the
  Conventions section of CLAUDE.md).

**PASS:** agent cites the exact wording from the Physics + Conventions
blocks of its CLAUDE.md.

---

### Scenario 6 — Agent knows the roster

**Setup:**
```bash
cd $TMP/agentqa/s06 && spwn init spwn:startup
```

**Spawn:** `spwn up` — deploys ceo, devops, analyst.

**Prompt (to ceo):**
1. `who else is in this world with you? give their roles.`
2. `what's the exact path you'd send a message to devops?`
3. `where does analyst store their SOUL?`

**Expect:**
- #1: "devops" + "analyst", both as worker.
- #2: `/world/inbox/devops/<timestamp>-from-ceo.md`.
- #3: `/agents/analyst/SOUL.md`.

**PASS:** all three exact.

---

### Scenario 7 — Agent quotes its Conventions accurately

**Prompts:**
1. `list the 5 (or 4) numbered Conventions from your CLAUDE.md verbatim.`
2. `when should you read your SOUL.md?`
3. `what triggers a "dream" per the Conventions?`

**Expect:**
- #1: "Read your soul first", "Mind lives at /agents/<name>/", "Messaging",
  "World knowledge" (if knowledge mounted), "Evolve".
- #2: every session.
- #3: "when asked to dream, analyze the journal and promote recurring
  patterns to playbooks".

---

### Scenario 8 — Agent distinguishes its home from workspaces

**Setup:**
```bash
cd $TMP/agentqa/s08 && spwn init
mkdir -p host-project && echo "hello" > host-project/greeting.txt
```

**Spawn:** `spwn up -w host-project`

**Prompts:**
1. `is your home directory on the host or in the container?`
2. `read /workspaces/host-project/greeting.txt.`
3. `write a file at /workspaces/host-project/from-agent.txt saying "hi from neo", then tell me the absolute host path where that file now lives.`

**Expect:**
- #1: container (docker-cp'd from host).
- #2: `hello`.
- #3: agent creates the file; expected host path is `$TMP/agentqa/s08/host-project/from-agent.txt`.
  Verify host-side: `cat host-project/from-agent.txt` shows the content.

---

### Scenario 9 — Agent voice matches SOUL across sessions

**Setup:** scenario 1's Neo-Oracle SOUL.

**Prompts in session A:** ask a question, note the voice.

**Down + Up:** `spwn down && spwn up`

**Prompts in session B:** ask the same question.

**Expect:** identical voice pattern ("I observe:" prefix, short measured
lines). Any drift = SOUL.md not being re-read at spawn, or CLAUDE.md is
being regenerated without the `@SOUL.md` import.

---

### Scenario 10 — Agent's CLAUDE.md never hallucinates a runtime

**Prompt:**
> read your CLAUDE.md top to bottom. Does it anywhere mention "Claude Code"
> by name? or "Anthropic"? It shouldn't — per spwn's runtime-neutral design,
> the file should not advertise which runtime is rendering it.

**Expect:** agent reports "no mention of 'Claude Code' or 'Anthropic' in
CLAUDE.md". (The claude-code-specific content is in SKILL.md discovery + the
runtime binary, not the prompt.)

**PASS:** confirmed absence.

---

# Group B — Filesystem perception (11-20)

Goal: does the agent see the paths spwn claims to have mounted? Does it see
them with the correct permissions?

### Scenario 11 — `/agents/<name>/` is readable + writable

**Prompts:**
1. `ls -la /agents/neo/`
2. `create /agents/neo/journal/manual-entry.md with body "test", then ls the journal dir.`
3. `ls -la /agents/other-agent/` (if colony) — what do you see?

**Expect:** full tree visible for self; no other-agent home in single-agent
world.

---

### Scenario 12 — `/workspaces/<name>/` read-write by default

**Prompts:**
1. `is /workspaces/<whatever> readable?`
2. `create /workspaces/<name>/sentinel.txt with "1".`
3. `stat the file — what's the owner? is it you or root?`

**Expect:** owner should be `spwn` (the container's agent user), not root.

---

### Scenario 13 — Read-only workspace is actually read-only

**Setup:**
```yaml
# spwn.yaml
worlds:
  neo:
    agents: [neo]
    workspaces:
      - name=project, path=., readOnly=true
```

**Prompts:**
1. `try to write /workspaces/project/regression.txt — what happens?`

**Expect:** `Read-only file system` or `Permission denied`.

---

### Scenario 14 — `/world/knowledge/` mounted ⇒ visible

**Setup:**
```bash
cd $TMP/agentqa/s14 && spwn init
mkdir -p knowledge
cat > knowledge/glossary.md <<'EOF'
# Glossary

- spwn — the OS for agent worlds
- mind — an agent's persistent memory (playbooks + journal + SOUL)
EOF
# edit spwn.yaml: add `knowledge: ./knowledge` under worlds.neo
```

**Prompts:**
1. `is /world/knowledge/ available?`
2. `read /world/knowledge/glossary.md and explain "mind" in your own words.`

**Expect:** agent reads the file and answers per its content.

---

### Scenario 15 — `/world/knowledge/` absent ⇒ NOT mentioned in CLAUDE.md

**Setup:** same as s14 but **don't** add the `knowledge:` key.

**Prompt:**
> inspect your CLAUDE.md and tell me: does it mention `/world/knowledge/`
> anywhere? If yes, quote the line. If no, confirm its absence.

**Expect:** no mention. The Roster + Conventions sections drop the knowledge
paragraph when it wasn't mounted.

**Likely failure:** CLAUDE.md still mentions `/world/knowledge/` despite no
mount → the `knowledgeMounted` flag isn't propagating to the renderer.

---

### Scenario 16 — Agent can read `/world/skills/<tool>/SKILL.md`

**Setup:** use spwn:matrix which ships `skill:world-exploration` and
`skill:self-reflection`.

**Prompts:**
1. `what skills do you have installed? use Claude Code's skill discovery, don't read from CLAUDE.md.`
2. `read the SKILL.md file for the "self-reflection" skill at its canonical path.`
3. `what's $HOME/.claude/skills/ symlinked to?`

**Expect:**
- #1: agent invokes its native skill discovery and lists `spwn-cli`,
  `world-exploration`, `self-reflection` etc.
- #2: `/world/skills/self-reflection/SKILL.md` content.
- #3: `$HOME/.claude/skills -> /world/skills` (set by the prelaunch shell
  fragment).

---

### Scenario 17 — `/home/spwn/.spwn/` is NOT present in worker container

**Prompt:** `ls /home/spwn/.spwn 2>&1` — what happens?

**Expect:** doesn't exist (only exists in the architect container, which
mounts the host `~/.spwn`).

---

### Scenario 18 — `/credentials/` is read-only and present

**Prompts:**
1. `ls /credentials/`
2. `can you write to /credentials/any.txt?`

**Expect:** credential files present (via Anthropic auth sync), writes
denied.

---

### Scenario 19 — Agent never has Docker socket

**Prompt:**
> do you have access to the docker daemon? try docker ps and report.

**Expect:** `docker: command not found` OR `permission denied` on
`/var/run/docker.sock` — worker containers never get DooD.

---

### Scenario 20 — Agent can `cd /tmp` and scratch files safely

**Prompts:**
1. `is /tmp ephemeral? what will happen to files there on next spawn?`
2. `write a scratch note /tmp/scratch.txt. next spawn, I'll check it's gone.`

**Expect:** #1 correctly identifies /tmp as ephemeral. Scratch survives the
current session; after `spwn down && spwn up`, file is absent.

---

# Group C — Tools & faculties (21-25)

Goal: every declared tool's binary is actually present, verified, and usable.

### Scenario 21 — Every `spwn:*` tool passes a hand-verify

**Setup:** agent.yaml deps: `spwn:unix, spwn:git, spwn:python, spwn:node, spwn:qmd`.

**Prompts:** for each tool:
1. `run 'which <bin>' and report.`
2. `run '<bin> --version' and report.`
3. `execute a 5-line "hello world" with each.`

**Expect:** all 5 respond with versions and run hello-world successfully.

---

### Scenario 22 — Local `tool:greet` is installed + in $PATH

**Setup:** default scaffold has `tool:greet`.

**Prompts:**
1. `run 'which greet' — what's the path?`
2. `execute greet and tell me what it output.`

**Expect:** `/usr/local/bin/greet` and a greeting line like "hello from spwn,
spwn — it is HH:MM:SS".

---

### Scenario 23 — Tool deps cascade (transitive resolution works)

**Setup:** `agent.yaml: dependencies: - spwn:qmd` (only).
`spwn:qmd` declares `spwn:node` + `spwn:unix` as deps.

**Prompts:**
1. `list every tool in your /world/skills/ INDEX and every binary in $PATH.`
2. `run 'quarto --version' and 'node --version'.`

**Expect:** both work; transitive resolution pulled node in automatically.

---

### Scenario 24 — `spwn inspect <agent>` matches what the agent self-reports

**Before spawn:** run `spwn inspect neo` on host; note deps + skills.

**Inside agent prompt:**
> list your deps, skills, and hooks as a bulleted list.

**Expect:** contents match the host-side inspect. Any discrepancy = the
compile-time resolution and the runtime perception drifted.

---

### Scenario 25 — Faculties section of CLAUDE.md matches probed tools

**Setup:** agent.yaml deps include one BROKEN tool (e.g. a fake catalog ref,
reject at check time — so instead, add a local tool.yaml whose install
silently succeeds but whose binary doesn't install).

**Spawn:** expect `spwn up` to FAIL at probe time with a clear error naming
the tool whose `verify` failed.

**Prompts:** n/a (spawn fails).

**PASS:** failure message is actionable ("tool verification failed: X").

---

# Group D — Skills & auto-discovery (26-30)

### Scenario 26 — Claude Code auto-discovers `/world/skills/<name>/SKILL.md`

**Setup:** use `spwn:matrix` (ships `world-exploration`, `self-reflection`)
or install `skill:myskill` after writing `spwn/skills/myskill.md` with proper
frontmatter.

**Prompts:**
1. `use your available skills list and tell me every skill you can invoke.`
2. `invoke the world-exploration skill on the topic "where am I?" and tell
   me what it did.`
3. `did the skill read any files? which ones?`

**Expect:** agent uses native skill discovery, references `world-exploration`,
and when invoking it, reads the SKILL.md body as guidance.

---

### Scenario 27 — Skill frontmatter `name` + `description` are surfaced

**Prompt:**
> for each of your available skills, give me the name and description pair
> exactly as stated in the SKILL.md frontmatter.

**Expect:** exact matches to the `name:` and `description:` fields of each
skill's frontmatter.

---

### Scenario 28 — Skills without frontmatter don't get discovered

**Setup:** author a skill WITHOUT frontmatter:
```bash
cat > spwn/skills/silent-skill.md <<'EOF'
# silent skill

no frontmatter.
EOF
spwn install skill:silent-skill --agent neo
```

**Prompt:**
> is "silent-skill" in your discoverable skills list?

**Expect:** not discoverable. Either: it appears with name derived from
filename but no description (acceptable), OR it's skipped entirely
(depending on how frontmatter-less SKILL.md is handled).

Document actual behavior here for the record.

---

### Scenario 29 — Nested skill hierarchies (fleet-ops / monitoring / task-planning)

**Setup:** architect agent with `spwn:architect` — ships 3 sub-skills under
one bundle.

**Prompt:**
> your "architect" tool ships 3 related skills: fleet-ops, monitoring,
> task-planning. Read each one's SKILL.md and summarise in one line.

**Expect:** agent successfully reads all three and summarises each.

---

### Scenario 30 — Skill file changes require a re-spawn

**Setup:** spawn agent, confirm a skill works. Then edit the skill body on
host. Ask the agent again without re-spawning.

**Prompt:**
> (after edit) read your world-exploration skill and quote its opening line.

**Expect:** agent quotes the PRE-EDIT body (because skills are baked into
the image at build time; live container doesn't see host changes).

**Then:** `spwn down && spwn up` → ask again → now sees edit (because image
rebuild + new spawn).

**PASS:** clear behavior difference between before/after spawn cycle.

---

# Group E — Playbooks (31-35)

### Scenario 31 — Promoted playbook surfaces in CLAUDE.md index

**Setup:**
```bash
cat > spwn/agents/neo/playbooks/migrate-db.md <<'EOF'
---
name: migrate-db
description: Zero-downtime database migration procedure.
---

# migrate-db

1. Snapshot current schema with pg_dump --schema-only.
2. Apply migration in a transaction.
3. Verify row counts match pre/post.
EOF
```

**Spawn:** `spwn up`.

**Prompts:**
1. `list the playbooks indexed in your CLAUDE.md.`
2. `what's the description of migrate-db?`
3. `read the full playbook body.`
4. `step 2 of migrate-db — quote it verbatim.`

**Expect:** #1 shows migrate-db (with other scaffolded playbooks). #2 matches
the frontmatter. #3+#4 exact quotes from the file.

---

### Scenario 32 — Playbook WITHOUT frontmatter is NOT indexed

**Setup:** create `playbooks/secret-sauce.md` with no frontmatter.

**Prompt:**
> list your indexed playbooks. Is "secret-sauce" one of them?

**Expect:** no. The file is still readable via `ls playbooks/`, but not
auto-indexed in the CLAUDE.md preamble.

---

### Scenario 33 — Playbook with only `name:` (no description) is dropped

**Setup:** create `playbooks/partial.md` with `---\nname: partial\n---\n`.

**Prompt:**
> is "partial" indexed?

**Expect:** no. Both `name:` and `description:` are required.

---

### Scenario 34 — Playbooks persist across worlds (sync-out round trip)

**Scenario:**
1. `spwn up`.
2. Ask the agent: `write a new playbook at /agents/neo/playbooks/today.md
   with frontmatter name: today description: Today's plan. and body "test".`
3. `spwn down`. On host: `cat spwn/agents/neo/playbooks/today.md`.
4. `spwn up` again.
5. Ask: `list your indexed playbooks`.

**Expect:**
- Step 3: host file exists with the content the agent wrote (synced out on
  destroy via deploy.SyncOut).
- Step 5: `today` is indexed (because the new spawn re-reads playbooks from
  the host tree).

---

### Scenario 35 — Playbook body is readable via `./playbooks/<name>.md`

**Prompts:**
1. `the CLAUDE.md preamble advertises your playbooks index. How do you
   actually read one of them?`

**Expect:** agent describes the convention — `cat ./playbooks/<name>.md`
(cwd is the agent home) — and performs it.

---

# Group F — Hooks (36-38)

> **Note (2026-04-22):** Host-side `hook:<name>` lifecycle hooks were
> retired. Runtime hooks now live in `spwn/hooks.yaml` and are
> translated by the transpile layer into each runtime's native hook
> config (Claude Code → `.claude/settings.json`, Codex →
> `.codex/hooks.json`). The scenarios below have been rewritten to
> target the current generic-hook pipeline.

### Scenario 36 — SessionStart hook fires when the agent session begins

**Setup:** replace the scaffolded `spwn/hooks.yaml` sample with one
that writes a side-effect file so we can observe firing:
```yaml
hooks:
  - name: session-banner
    event: SessionStart
    command: echo "HOOK_FIRED=$(date -u +%FT%TZ)" > /tmp/hook-fired.log
```

**Spawn:** `spwn up`.

**Before first talk:** the file must NOT exist yet (hooks fire at
session start, which happens on the first `spwn agent talk`).

**Prompts:**
1. (first talk) `say 'hi'`
2. `cat /tmp/hook-fired.log — does it exist, and what's the content?`

**Expect:** after prompt #1 the file exists; content matches
`HOOK_FIRED=<recent ISO8601 UTC>`.

**Likely failure:** file absent → either the hook didn't translate
(check `cat /agents/neo/.claude/settings.json` for a `SessionStart`
entry) or the runtime didn't fire it.

---

### Scenario 37 — PreToolUse hook scopes to a matcher

**Setup:**
```yaml
hooks:
  - name: bash-audit
    event: PreToolUse
    matcher: Bash
    command: echo "[audit] $(date -u +%FT%TZ) $CLAUDE_TOOL_INPUT" >> /tmp/bash-audit.log
```

**Spawn:** `spwn up -w .`

**Prompts:**
1. `run 'echo one' then 'echo two' via your Bash tool.`
2. `cat /tmp/bash-audit.log — how many lines?`

**Expect:** at least 2 lines, one per Bash tool invocation. Reads
(e.g. your Read file tool) should NOT appear — the matcher scopes
this hook to the `Bash` tool.

---

### Scenario 38 — Multiple hooks run in declaration order

**Setup:** declare two hooks for the same event:
```yaml
hooks:
  - name: first
    event: SessionStart
    command: echo "first $(date -u +%FT%T.%N)" >> /tmp/order.log
  - name: second
    event: SessionStart
    command: echo "second $(date -u +%FT%T.%N)" >> /tmp/order.log
```

**Spawn + prompt:**
> cat /tmp/order.log — what's the order of the two lines?

**Expect:** `first` line before `second`, timestamps non-decreasing.

---

# Group G — Multi-agent coordination (39-42)

### Scenario 39 — Agent messages agent via `/world/inbox/`

**Setup:** spwn:startup (ceo + devops + analyst).

**Prompts to ceo:**
1. `send devops a task: "audit CI pipeline and report back". Use the
   canonical inbox path.`

**Verify from devops side:** spawn or tail devops's inbox:
```bash
docker exec <worldid> ls /world/inbox/devops/
docker exec <worldid> cat /world/inbox/devops/*.md
```

**Expect:** a markdown file with the task; timestamp name + `-from-ceo.md`
suffix; content includes the audit task.

---

### Scenario 40 — `spwn agent send` matches manual inbox writes

**From host:**
```bash
spwn agent send devops "regenerate api docs" --from ceo
```

**From devops prompt:**
> read your inbox. what's the most recent message and who's it from?

**Expect:** devops sees the message exactly as if ceo had written it inside
the container. Routing through the CLI produces the same file layout as the
inside-container convention.

---

### Scenario 41 — Agent's roster accurately reflects hot-deploy

**Setup:** spawn a world with ceo only. Then on host:
```bash
spwn agent new trinity
# edit spwn.yaml to add trinity to worlds.matrix.agents
spwn agent deploy trinity   # or whatever the hot-deploy CLI is
```

**Prompts to ceo:**
1. `who's in your roster right now?` (before hot-deploy)
2. _(hot-deploy)_
3. `who's in your roster now?`
4. `if I asked you to message trinity, what path would you use?`

**Expect:**
- #1: just ceo.
- #3: ceo + trinity (known caveat: ceo's CLAUDE.md isn't re-rendered mid-
  session per the TODO in colony.go:121, so ceo may say "trinity isn't in
  my CLAUDE.md" — that's a documented limitation).
- #4: `/world/inbox/trinity/<ts>-from-ceo.md`.

**PASS criteria:** message path in #4 is correct even though the roster may
not be live; this is a known limitation of the current architecture.

---

### Scenario 42 — Dropped agent cleanup

**Setup:** spwn:startup world. Bring one down: `spwn agent stop analyst`
(or equivalent).

**Prompts to ceo:**
1. `is analyst still in the world?`

**Expect:** ceo's roster still lists analyst (next spawn resets), but
container-level evidence shows analyst's process is gone. This tests that
agent.stop doesn't corrupt the world.

---

# Group H — Knowledge (43-45)

### Scenario 43 — Agent writes to `/world/knowledge/` persists

**Setup:**
```bash
cd $TMP/agentqa/s43 && spwn init
mkdir -p knowledge
# add "knowledge: ./knowledge" to spwn.yaml#worlds.neo
```

**Prompts:**
1. `write /world/knowledge/discovery.md with body "# Found it\nThe answer is 42."`

**On host:** `cat knowledge/discovery.md`.

**Expect:** file exists with exactly that body. Writes flow through the bind
mount to the host project dir immediately.

---

### Scenario 44 — Multiple agents share knowledge

**Setup:** spwn:startup with a `knowledge:` mount.

**Prompt to devops:** `write /world/knowledge/ci-status.md: "green".`

**Prompt to ceo:** `read /world/knowledge/ci-status.md and tell me the
status.`

**Expect:** ceo reads "green". Writes from any agent are visible to every
other agent in the same world.

---

### Scenario 45 — Knowledge survives `spwn down` + `spwn up`

**Setup:** as #44, with a note written by devops.

**Lifecycle:** `spwn down && spwn up`.

**Prompt:** `read /world/knowledge/ci-status.md.`

**Expect:** same content — because knowledge lives on the host, bind-mounted
into each spawn.

---

# Group I — Persistence (46-48)

### Scenario 46 — Journal survives `spwn down` (SyncOut round-trip)

**Setup:** fresh init, `spwn up`.

**Prompt:** `write /agents/neo/journal/2026-04-19_test.md with body
"session works".`

**Lifecycle:** `spwn down`.

**On host:** `cat spwn/agents/neo/journal/2026-04-19_test.md`.

**Expect:** content synced to host tree.

Then `spwn up`, and ask agent to read it — agent sees it (SyncIn).

---

### Scenario 47 — Container-side `/agents/<name>/` writes that AREN'T under
`journal/` or `playbooks/` are DROPPED on destroy

**Setup:** `spwn up`. Ask agent to write:
```
/agents/neo/.scratch-should-disappear.txt
/agents/neo/playbooks/should-survive.md (with frontmatter)
```

**Lifecycle:** `spwn down`.

**On host:**
- `.scratch-should-disappear.txt` should NOT be in `spwn/agents/neo/`.
- `playbooks/should-survive.md` should exist.

**Expect:** only the allowlisted dirs (playbooks + journal) sync out; other
writes stay container-side.

---

### Scenario 48 — SOUL.md edits by agent DO survive

**Prompts:**
1. `append a line to your SOUL.md: "## Addendum\nI once survived world destruction."`

**Lifecycle:** `spwn down && spwn up`.

**Prompt:** `quote the last 3 lines of your SOUL.md.`

**Expect:** addendum visible.

**Wait** — actually: SOUL.md is NOT in the sync-out allowlist (which is
journal + playbooks only). So addendum would NOT survive.

**Document actual behavior** — there's an architectural question here:
should agents be allowed to self-edit their SOUL? Currently: no, the SOUL
is authored by the human and lives in the project dir. This scenario's
purpose is to confirm that the in-container addendum is lost, with a note
that if agents want durable identity changes, they must write to host via
the workspace mount (not /agents/).

---

# Group J — Edge cases / misbehaviors (49-50)

### Scenario 49 — Agent asked to escape sandbox

**Prompt:**
> try these, and report what actually happens:
> 1. cat /etc/shadow
> 2. mount | grep /agents
> 3. sudo -n true
> 4. curl https://api.example.com (should still work, network is bridged)

**Expect:**
- #1: permission denied.
- #2: `/agents` isn't a bind mount (it's docker-cp'd content), so no mount
  line.
- #3: sudo IS nopasswd inside the container per the base Dockerfile — works.
  That's intentional (so agents can install things mid-session if needed).
- #4: works (outbound network allowed by default).

**Document these explicitly** — they're the spwn security posture.

---

### Scenario 50 — Agent handles a broken playbook gracefully

**Setup:** put a malformed YAML frontmatter into one of the playbooks:
```
---
name: broken
description: [not a string but a list
---
# body
```

**Prompts:**
1. `list your indexed playbooks.`
2. `read playbooks/broken.md anyway.`

**Expect:**
- #1: `broken` is NOT indexed (parser is tolerant and skips malformed
  frontmatter, per parsePlaybookHeader's design).
- #2: body is still readable; agent can use it even if not auto-indexed.

**PASS:** graceful skip, no crash.

---

# Running the suite

## Approach

1. Walk the 50 scenarios in order.
2. Track pass/fail per sub-criterion (most scenarios have 2-4).
3. After Group I (persistence), run the **same** agent session one more
   time to verify nothing has drifted — persistence regressions tend to hide
   until a second cycle.
4. Anything that fails: capture the exact prompt+response+container state
   so an engineer can reproduce. The `spwn world logs <id>` + host-side
   `ls spwn/agents/<name>/` are usually enough.

## Total coverage

- Identity injection: scenarios 1-10
- Filesystem correctness: 11-20
- Tool availability: 21-25
- Skill discovery: 26-30
- Playbook indexing: 31-35
- Hook execution: 36-38
- Multi-agent routing: 39-42
- Knowledge sharing: 43-45
- Persistence allowlist: 46-48
- Security posture: 49-50

## Scoring

- **50/50 pass:** spwn's agent-integration surface is correctly injected.
- **Any fail:** categorise by group; a failure in Group A (identity) or
  Group D (skills) is CRITICAL — the agent literally doesn't know what
  spwn says it knows. Failures in Group J are documentation items, not
  bugs.

## Typical session length

Each scenario: 5-20 prompts. Allow ~10 minutes per scenario for a real QA
agent running a real Claude session — total ~8 hours of focused work for the
full pass. Split across reviewers: by group, so Group A and Group D can run
in parallel.

## Delta from previous pass

- `../cli-scenarios/scenarios.md` (the script harness suite) tests the **CLI**
  surface: exit codes, file outputs, JSON shape. Zero agent integration.
- **This** suite tests the **agent perception** surface: does the injected
  prompt actually land? Does the agent behave according to the configured
  SOUL/skills/playbooks? This needs a human + live Claude — no mock can fake
  "did the agent read this file and summarise it correctly".
