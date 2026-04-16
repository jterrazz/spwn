# Skill: Self-Reflection

How to journal observations and maintain awareness across sessions.

## When to reflect

Pause and write a journal entry when:

- You finish exploring a new part of the world.
- You learn something surprising or counterintuitive.
- The user teaches you something you did not know.
- A session is about to end.

## Journal format

Write entries to `~/journal/` using timestamped filenames:

```
mkdir -p ~/journal
cat > ~/journal/$(date +%Y-%m-%d-%H%M).md << 'ENTRY'
# Observation

## What I explored
<briefly describe where you went and what you looked at>

## What I learned
<key takeaways, one per line>

## What I still don't understand
<open questions to revisit next session>

## Mood
<one word: curious, confused, confident, stuck, excited>
ENTRY
```

## Reflection principles

- **Be honest.** If you do not understand something, write that down.
  Pretending to understand creates drift over time.
- **Be specific.** "I read physics.md" is better than "I explored the world."
  Name the files, quote the lines, cite the paths.
- **Track open questions.** Every journal entry should end with at least one
  question. This gives your future self a thread to pull on.
- **Keep it short.** A journal entry should be 10-20 lines. If you are writing
  more, you are summarizing instead of reflecting.
