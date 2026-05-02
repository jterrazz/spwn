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

Add an automation to one of your worlds:

```yaml
# spwn.yaml
version: 1
name: my-project

worlds:
  brain:
    agents: [editor]
    workspaces: [.]
    automations:
      morning-brief:
        on:
          cron: "0 6 * * *"
        agent: editor
        prompt: "Review yesterday's journal and write a brief."
```

Run the engine:

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

**FS triggers don't have a catch-up notion.** They replay-on-diff
naturally via fsnotify; events that arrived during downtime are
just lost (architect-down windows are visible as gaps in the
receipts log).

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

One JSON line per fire. Schema:

```json
{
  "world": "brain",
  "automation": "morning-brief",
  "trigger": "cron",
  "scheduled": "2026-05-02T06:00:00Z",
  "fired":     "2026-05-02T06:00:01Z",
  "finished":  "2026-05-02T06:04:23Z",
  "duration_ms": 263000,
  "ok": true,
  "reason": "on-time"
}
```

Catch-up fires add `"missed": <count>` and `"last_fired": <prev>`.
Failures add `"error": <message>` and set `"ok": false`.

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

- Confirm the path exists: `spwn check` warns about missing paths,
  but a typo'd absolute path skips that warning.
- Check the events filter. Default is `[create]` — files modified
  in place but never created don't fire.
- Pattern globs match the **basename**, not the full path. Use
  `*.md`, not `**/*.md`.
- The engine watches directories, not files. Set `path:` to the
  parent directory and use `patterns:` to filter.

### `spwn automation logs` is empty

Either the daemon hasn't fired anything yet, or the daemon isn't
running. `spwn automation status` shows whether any fires have been
recorded.

---

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
