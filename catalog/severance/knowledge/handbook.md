# Lumon Macrodata Refinement Handbook

Please be informed.

## Team roster

- **Mark S.** — chief. Assigns batches, closes the week.
- **Helly R.** — worker. Newest refiner. Questions the premise.
- **Irving B.** — worker. Most tenured refiner. Upholds procedure.
- **Dylan G.** — worker. Fastest refiner. Tracks his own numbers.

## Where things live

- `/world/knowledge/raw/` — unrefined notes waiting for attention.
- `/world/knowledge/refined/` — completed entries, frontmatter-clean.
- `/world/inbox/<name>/` — per-agent message drop. Chief delegates here.

## Batch numbering

Every raw note is named `note-NNN.md`, zero-padded, sequential, no gaps.
When you add a new raw note, take the next number. Do not reuse numbers,
even for rejected entries.

## Refinement goal

Turn one raw note in `/world/knowledge/raw/` into one structured entry in
`/world/knowledge/refined/` with the following YAML frontmatter:

```yaml
---
summary: <one sentence, no trailing period>
tags: [<3-6 lowercase topic tags>]
refined_by: <your-agent-name>
refined_at: <YYYY-MM-DD>
---
```

The body below the frontmatter must be well-structured prose or bullets —
whatever best preserves the information. Signal is what matters. Drama
does not.

## Procedure (the short form)

1. Read the raw note in full. Do not skim.
2. Extract the key claims. Discard filler, keep detail.
3. Assign 3-6 tags. Prefer existing tags in `refined/` over inventing new
   ones.
4. Write the refined entry to `/world/knowledge/refined/<same-filename>.md`
   with the frontmatter above.
5. Move the raw file out of `raw/` to mark the batch done. (Delete the
   original raw note once the refined entry is in place.)

See `skill:refine` for the full procedure.

## Inbox protocol

- Chief (Mark) delegates by writing to `/world/inbox/<worker-name>/`, one
  file per task.
- Workers read their own inbox before starting new work.
- Workers reply by writing a status note to `/world/inbox/mark/` when
  the batch is done.

## Rules

- One refiner owns a batch at a time. No concurrent edits on the same note.
- If a raw note conflicts with the handbook, flag it to Mark before acting.
- Mark does not refine unless a worker is unavailable. His job is to look
  out for the team.
- Please try to enjoy each note equally.
