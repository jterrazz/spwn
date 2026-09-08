# Automations

Automations bind a **trigger** to one of your project's agents. A
trigger fires (a cron tick, a new file in a watched directory) → the
architect wakes the target agent → the rendered prompt is delivered.

Two trigger kinds in v1:

- **`cron:`** — fire at a wall-clock schedule. 5-field standard cron
  expression evaluated against the host clock.
- **`fs:`** — fire on filesystem events under a watched directory.
  Backed by fsnotify; no polling.

This page is the user guide. For the public Go API see
[`packages/automation`](../packages/automation/automation.go).

---

## Quickstart

**Prerequisites.** Automations dispatch INTO a running world's
container. Before you write your first automation, scaffold the
project end-to-end:

```bash
spwn init                    # creates spwn.yaml + spwn/agents/
spwn agent new editor        # creates spwn/agents/editor/
spwn check                   # confirm the tree is valid
spwn up                      # bring up the world (architect spawns containers)
```

Once you have a running world, add the automation block to your
`spwn.yaml` (the example shows a minimal entry; merge into the
existing worlds map):

```yaml
# spwn.yaml
version: 1
name: my-project

worlds:
  brain:                                # the world `spwn up` brought online
    agents: [editor]
    workspaces: [.]
    automations:
      morning-brief:
        on:
          cron: "0 6 * * *"
        agent: editor
        prompt: "Review yesterday's journal and write a brief."
```

Verify with `spwn check` (parses the new block) and `spwn automation ls`
(shows the entry plus its placeholder `LAST FIRED`):

```
$ spwn check
✓ Project is valid
$ spwn automation ls
WORLD  NAME           TRIGGER         AGENT   LAST FIRED
brain  morning-brief  cron 0 6 * * *  editor  —
```

Then run the engine:

```
$ spwn automation daemon
automation daemon — project my-project
registered automations: 1
receipts → /Users/jterrazz/my-project/.spwn/runs.jsonl
ctrl-c to stop
```

The daemon blocks. Every time a trigger fires, a row is appended to
`.spwn/runs.jsonl` and the agent receives the prompt inside its
running world.

> **Heads up:** `spwn automation daemon` does NOT bring worlds up.
> Run `spwn up <world>` first; otherwise fires record as failed
> receipts ("no running world found"). The daemon is stateless
> beyond its receipts — it loads the manifest at boot, so any edit
> to `spwn.yaml` requires a daemon restart to take effect.

---

## Schema

Each entry under `worlds.<world>.automations.<name>` is one
automation. The schema:

```yaml
automations:
  <name>:                          # kebab-case slug; must match ^[a-z][a-z0-9-]*$
    on:                            # exactly one of cron / fs
      cron: "<5-field expression>"
      fs:
        path: <directory>          # absolute or project-relative
        events: [create, write, rename]   # default: [create]
        recursive: <bool>          # default: false
        debounce: <duration>       # default: 1s; min 100ms, max 1h
        patterns: ["*.md", "*.pdf"] # default: match all
        include_hidden: <bool>     # default: false; recursive only
    agent: <name>                  # must be in worlds.<world>.agents
    prompt: <inline text>          # XOR with command:
    command: command/<name>        # XOR with prompt:; resolves to spwn/commands/<name>.md
    catchup: collapse | skip       # cron only; default collapse
```

**Body:** exactly one of `prompt:` (inline string) or `command:` (a
`command/<name>` ref pointing at `spwn/commands/<name>.md`).
Reusing a slash-command file means the same prompt can be invoked
three ways: by the agent via `/morning-brief`, manually via
`spwn agent talk`, or automatically via the cron.

**Trigger:** exactly one of `cron:` or `fs:`. `spwn check` rejects
both-set / neither-set.

**Agent:** must be one of the world's declared agents — the engine
talks to the agent inside the world's container.

---

## Catch-up

What happens when the architect was down across one or more
scheduled cron slots? Two modes, mirroring Apple Reminders semantics:

| Mode       | Behaviour                                                                  |
|------------|----------------------------------------------------------------------------|
| `collapse` | One fire on resume regardless of how many slots were missed (default).     |
| `skip`     | No fire on resume. The schedule resumes at the next scheduled slot.        |
| `stack`    | One fire per missed slot, in order, capped at 100 to bound blast radius.   |

In `collapse` mode the prompt template can reference the missed
count via `{{ .Missed }}` and the previous successful fire via
`{{ .LastFired }}`:

```yaml
prompt: |
  Brief.
  {{ if .Missed }}({{ .Missed }} slots missed since {{ .LastFired | date "2006-01-02" }}){{ end }}
```

The receipt for a catch-up fire records both the slot it covered
(`scheduled`) and when it actually ran (`fired`) — the dashboard can
render "ran 2h late" trivially.

**No catch-up on first boot.** If the engine has no record of a
prior fire (no entry in `.spwn/automations/state.json`), it never
fires a catch-up. The engine doesn't fabricate history.

**FS triggers also catch up.** On daemon start, every fs trigger
walks its watched directory and fires once for files whose mtime
is newer than the last successful fire. Same `catchup: skip` opt-out
as cron. Use cases: laptop asleep while a partner drops files into
the inbox; daemon restart during a deploy.

The reason for replay fires is `replay:<basename>` (vs `create:<basename>`
for live fsnotify events). The full path list lands in the
receipt's `event_paths` field — your prompt template can iterate
with `{{ range .Event.Paths }}…{{ end }}`.

**DST gaps are silently dropped.** Cron expressions evaluate against
the host's local time. On the spring-forward day in regions that
observe DST, the wall-clock hour skipped (e.g. 02:00→03:00 in US
Eastern) is missing entirely — `cron: "0 2 * * *"` will not fire on
the gap day, and no catch-up is recorded for the missing slot. If
this matters, schedule the cron at a non-gap hour (`0 3 * * *`) or
express it in UTC by running the daemon with `TZ=UTC`.

**Time-zone footgun.** The engine evaluates schedules in the time
zone of the recorded last-fired cursor. If the host's `TZ` changes
between fires (laptop travel, server reconfiguration), slots may
land at unexpected wall-clock hours until the cursor is rewritten
by a successful fire. Run the daemon with a fixed `TZ` if cron
slots must be stable across moves.

---

## Templating

Bodies are Go `text/template` strings. Available variables:

| Variable           | When set                              |
|--------------------|---------------------------------------|
| `{{ .Now }}`       | Always — wall time of the fire.       |
| `{{ .Scheduled }}` | Always — the slot this fire covers.   |
| `{{ .Reason }}`    | Always — `"on-time"`, `"catchup"`, or `"create:foo.md"` for fs. |
| `{{ .Missed }}`    | Catch-up cron only. Count of collapsed slots. |
| `{{ .LastFired }}` | Catch-up cron only. Previous successful fire's scheduled time. |
| `{{ .Event.Path }}`  | FS only. First path of the burst.   |
| `{{ .Event.Name }}`  | FS only. Basename of `Path`.        |
| `{{ .Event.Paths }}` | FS only. Full path list (debounce coalesces). |
| `{{ .Event.Kind }}`  | FS only. `create` / `write` / `rename`. |

A `date` helper formats times with Go's reference layout:

```yaml
prompt: 'Brief for {{ .Now | date "2006-01-02" }}.'
```

Templates that reference `.Event` from a cron automation render
empty (not nil-deref): use `{{ if .Event }}…{{ end }}` if your
template runs both ways.

---

## On-disk artefacts

Two files per project:

```
<project>/.spwn/
├── runs.jsonl                      # receipt log, append-only
└── automations/
    └── state.json                  # last-fired cursors per automation
```

### `runs.jsonl`

One JSON line per fire. Rotates at 100MB to `runs.jsonl.1` (with
`.1`→`.2`, etc., keeping 5 historical files). Total disk footprint
per project is bounded at ~500MB.

Schema (every field is timestamped UTC, sortable as plain text):

```json
{
  "world": "brain",
  "automation": "morning-brief",
  "agent": "editor",
  "trigger": "cron",
  "run_id": "9f3a7b2e8c1d4502",
  "engine_version": "spwn-automation/1",
  "scheduled": "2026-05-02T06:00:00Z",
  "fired":     "2026-05-02T06:00:01Z",
  "finished":  "2026-05-02T06:04:23Z",
  "duration_ms": 263000,
  "ok": true,
  "reason": "on-time"
}
```

| Field            | Always set | Notes |
|------------------|---|---|
| `world`          | ✓ | Manifest world key |
| `automation`     | ✓ | Manifest automation key |
| `agent`          | ✓ | Saves a join when grouping by agent |
| `trigger`        | ✓ | `cron` or `fs` |
| `run_id`         | ✓ | 16-hex unique per fire; pair with structured-logger output |
| `engine_version` | ✓ | Schema generation; dashboards branch on this |
| `fired`          | ✓ | When the engine started dispatch |
| `finished`       | ✓ | When dispatch returned |
| `duration_ms`    | ✓ | `finished - fired`, pre-computed |
| `ok`             | ✓ | true iff dispatch succeeded |
| `reason`         | ✓ | `on-time` / `catchup` / `replay:<file>` / `create:<file>` |
| `scheduled`      | when present | Cron slot the fire covered |
| `missed`         | catch-up only | Slots collapsed into this fire |
| `last_fired`     | catch-up only | Previous successful fire's scheduled |
| `error`          | failure only | Dispatcher's verbatim error string |
| `event_paths`    | fs only | Full path list of a debounce burst |
| `event_kind`     | fs only | Dominant op (`create` / `write` / `rename`) |
| `output`         | when present | Truncated stdout+stderr from the runtime exec (max 8KB) |
| `prompt_sha`     | success only | First 12 hex chars of sha256(prompt). "Did the prompt change between fires?" |
| `enqueued_at`    | success only | When the fire path entered (before the per-agent lock). `fired - enqueued_at` = lock wait |

### `state.json`

Per-`<world>/<name>` cursor of the last successful fire's
**scheduled** time (not the wall-clock fired time — a catch-up that
runs 2h late advances the cursor to the slot it covered, preventing
the next catch-up from re-detecting it).

Both files are gitignored by `spwn init`'s default `.gitignore`.

---

## CLI

```
spwn automation ls           — list every automation + last-fired
spwn automation status       — per-automation rollup (fires/ok/fail)
spwn automation logs [-n N]  — last N receipts
spwn automation logs -f      — tail receipts as they arrive
spwn automation daemon       — run the engine; ctrl-c to stop
```

`ls` / `status` / `logs` are read-only against `.spwn/`. They work
whether or not the daemon is running — `logs -f` will wait for the
file to appear if there's nothing yet.

---

## Cookbook

### Daily morning brief

```yaml
worlds:
  brain:
    agents: [editor]
    workspaces: [.]
    automations:
      morning:
        on: { cron: "0 6 * * *" }
        agent: editor
        prompt: |
          Brief for {{ .Now | date "2006-01-02" }}.
          {{ if .Missed }}({{ .Missed }} slots missed since {{ .LastFired | date "2006-01-02" }}){{ end }}

          Read /world/knowledge/feeds/ and write today's brief to
          /world/knowledge/briefs/{{ .Now | date "2006-01-02" }}.md.
```

### Watch an inbox folder

```yaml
worlds:
  brain:
    agents: [curator]
    workspaces: [.]
    automations:
      inbox-pull:
        on:
          fs:
            path: ./inbox
            events: [create]
            recursive: true
            debounce: 10s
            patterns: ["*.md", "*.pdf"]
        agent: curator
        prompt: |
          New file at {{ .Event.Path }}.
          Read it, decide whether to file under /world/knowledge or
          leave for human review.
```

### Reusable command body

`spwn/commands/process-inbox.md`:

```markdown
You're processing a new inbox arrival.

File: {{ .Event.Path }}
Kind: {{ .Event.Kind }}

1. Read the file.
2. Tag it (research / personal / admin / reference).
3. Move it to /world/knowledge/<tag>/.
4. Update /world/knowledge/index.md with a one-line entry.
```

`spwn.yaml`:

```yaml
automations:
  inbox-pull:
    on: { fs: { path: ./inbox } }
    agent: curator
    command: command/process-inbox      # ← same body as the slash command
```

The agent can also invoke the same prompt manually via
`/process-inbox` inside its conversation, or you can run it once via
`spwn agent talk curator "$(cat spwn/commands/process-inbox.md)"`.

### Hourly tool-cache refresh

```yaml
worlds:
  brain:
    agents: [scout]
    workspaces: [.]
    automations:
      x-snapshot:
        on: { cron: "0 */4 * * *" }    # every 4h
        agent: scout
        catchup: skip                   # don't backfill if I was offline
        prompt: |
          Refresh the X bookmark + likes snapshot for the last 4h.
          Run `make x-snapshot` and commit fresh data to
          /world/knowledge/feeds/.
```

---

## Troubleshooting

### `world "brain" is in status "stopped" (auto-spawn not yet implemented)`

The dispatcher requires the target world to be running. Bring it up
with `spwn up` or `spwn world up brain`. Auto-spawn-when-cold is on
the roadmap but not in v1; a fire against a stopped world records a
failed receipt and the next fire retries.

### `command file not found at spwn/commands/<name>.md`

Either the file is missing on disk, or the ref is misspelled in
`spwn.yaml`. `spwn check` catches both before runtime.

### `debounce 50ms is below the 100ms minimum`

Filesystem watchers below 100ms coalesce every keystroke. The engine
rejects sub-100ms windows at validation time.

### My cron fires every minute when I expected once a day

Cron's `*` means "every", not "once". `0 6 * * *` is "6:00 every
day" (minute=0, hour=6). `* * * * *` is "every minute, every hour,
every day". When in doubt, paste your expression into
[crontab.guru](https://crontab.guru).

### My fs trigger doesn't fire

- Confirm the path exists: `spwn check` blocks on missing paths
  with an explicit error.
- Check the events filter. Default is `[create]` — files modified
  in place but never created don't fire.
- The engine watches directories, not files. Set `path:` to the
  parent directory and use `patterns:` to filter.
- **Pattern syntax is `filepath.Match`, not shell glob.** Brace
  expansion (`*.{md,txt}`) and doublestar (`**/*.md`) are NOT
  supported and `spwn check` rejects them. Use multiple patterns
  for the brace case (`["*.md", "*.txt"]`) and combine
  `recursive: true` + a basename pattern for the doublestar case
  (`recursive: true, patterns: ["*.md"]`).
- Files added while the daemon was down are NOT replayed. fsnotify
  only sees events from after `daemon` started.

### Editor saves fire two receipts per save

Most editors save atomically: write `foo.md.tmp`, then rename to
`foo.md`. fsnotify reports a Create on both files. Filter out the
temp file with patterns:

```yaml
patterns: ["*.md", "!*.tmp", "!.*.swp"]   # last two NOT supported — see below
```

Negation isn't supported by `filepath.Match` either. The pragmatic
workarounds:

- Use a debounce window (default 1s) wide enough that both events
  collapse into a single fire — the rendered prompt's `.Event.Paths`
  lists every file touched in the burst.
- Have the agent inspect `.Event.Paths` and skip swap/temp files
  itself.

### `spwn automation logs` is empty

Either the daemon hasn't fired anything yet, or the daemon isn't
running. `spwn automation status` shows whether any fires have been
recorded.

---

## Limitations (v1)

- **One automation = one agent.** Multi-step pipelines (e.g. run
  agent A, then B with A's output, then C with both) cannot be
  expressed as a single automation in v1. Two workarounds for now:
  (a) have the trigger fire a coordinator agent that delegates to
  others via `Task`/`Bash`; (b) keep an existing `make` target as
  the orchestrator and let the automation just fire the pipeline.
  See `Pipelines (v2)` below for the planned schema.
- **No hot-reload.** The engine loads the manifest at `Start` and
  registers triggers once. Editing `spwn.yaml` while the daemon
  runs has no effect until restart.

The daemon takes an exclusive `flock` on
`<project>/.spwn/automations/daemon.lock` for its lifetime, so a
second `spwn automation daemon` for the same project fails fast
instead of interleaving receipts.

## Pipelines (v2, planned)

Sequential multi-agent pipelines are scheduled for the next pass.
Sketch of the planned schema for forward planning:

```yaml
automations:
  morning-newsroom:
    on: { cron: "0 6 * * *" }
    pipeline:
      - name: cluster
        agent: themer
        prompt: "Cluster yesterday's signal..."
      - name: assign
        agent: editor
        prompt: |
          Themer output: {{ .Steps.cluster.Output }}
          Now assign sections to writers.
      - name: write
        agent: writer
        prompt: "Editor's assignments: {{ .Steps.assign.Output }}"
```

Each step would be one fire (its own receipt), all sharing the
same `run_id`. Prior step outputs become available via
`{{ .Steps.<name>.Output }}` in subsequent prompts. Error policy:
abort on first failed step (or `continue: true` per-step opt-in).
Parallel steps + branching may follow if usage warrants. Until
v2 ships, the workarounds in the limitation above are first-class
solutions, not stopgaps.

## What's next

The current design intentionally punts on three things; tracked so
they don't fall through the cracks:

1. **Auto-spawn-when-cold.** A cron fire against a stopped world
   should bring the world up automatically rather than failing the
   receipt. Needs the manifest plumbed deeper into the dispatcher.
2. **Bind-mount path translation.** `{{ .Event.Path }}` currently
   shows the host path. Translating to the container's view at
   render time using `world.Workspaces` is straightforward; it just
   hasn't been wired yet.
3. **Webhook + message triggers.** `on: webhook:` and
   `on: message:` are reserved in the schema for future
   agent-to-agent fanout — once the message bus lands they slot
   into the same engine without API changes.
