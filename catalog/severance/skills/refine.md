---
name: refine
description: Turn a raw note in /world/knowledge/raw/ into a structured entry in /world/knowledge/refined/
---

# Skill: Refine

Turn a raw, messy note in `/world/knowledge/raw/` into a structured,
searchable entry in `/world/knowledge/refined/`. One raw file in, one
refined file out. Do not combine batches.

## When to use

Invoke this skill when you pick up a batch — either from your inbox
(`/world/inbox/<your-name>/`) or by claiming the next unrefined note
in `/world/knowledge/raw/`.

## Procedure

### 1. Read

Read the target raw note in full. Do not skim. If you do not understand
what the author was reaching for, re-read before touching the keyboard.

### 2. Extract

Pull out the key claims, decisions, numbers, names, and dates. Drop:

- filler ("ok so", "anyway", "idk")
- repeated points — keep the clearest phrasing
- side-tangents unrelated to the main thread (but note them as
  follow-ups if they're worth preserving)

Keep: concrete facts, dates, names, numbers, decisions, open questions.

### 3. Tag

Assign 3-6 lowercase tags. Check `/world/knowledge/refined/` first and
reuse existing tags wherever they fit. Invent a new tag only when no
existing one covers the topic.

Good tags are nouns or topic areas ("budget", "infrastructure",
"home-maintenance"). Bad tags are adjectives or moods ("important",
"urgent").

### 4. Write

Write the refined entry to `/world/knowledge/refined/<same-basename>.md`
with this frontmatter:

```yaml
---
summary: <one sentence, no trailing period>
tags: [<3-6 lowercase topic tags>]
refined_by: <your-agent-name>
refined_at: <YYYY-MM-DD>
---
```

The body below the frontmatter should be well-structured prose or bullet
lists — whatever best preserves the information. Prefer sections with
clear headings when the note covers more than one topic.

### 5. Close the batch

Once the refined entry is written, delete the original raw file so the
batch is visibly done. If you're a worker, leave a one-line status in
`/world/inbox/mark/` noting which note you closed.

## Quality checklist

- [ ] Frontmatter has all four fields and valid YAML.
- [ ] `summary` is one sentence that captures the point of the note.
- [ ] Tags are lowercase, 3-6, prefer-existing.
- [ ] Every concrete fact from the raw note survives in the refined one.
- [ ] No filler. No "ok so". No stream-of-consciousness.
- [ ] Raw file has been removed.
