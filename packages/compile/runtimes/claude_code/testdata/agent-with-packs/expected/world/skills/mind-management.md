# Mind Management

## Reading Your Soul
Before starting any task, read your SOUL.md — it carries your purpose,
voice, and principles. This is the single source of truth for who you
are.
```bash
cat /mind/SOUL.md
```

## Saving Knowledge
When you discover something worth remembering about the project or its
domain, write it to the world's knowledge base:
```bash
# Create a knowledge file with a descriptive name
echo "# What I learned about X" > /world/knowledge/topic-name.md
```
Knowledge is world-scoped: it's committed with the project and every
agent in this world sees the same files. Keep each file focused on
ONE topic and use clear filenames.

## Creating Playbooks
When you find a reusable procedure:
```bash
echo "# How to Deploy" > /mind/playbooks/deploy.md
# Include: trigger conditions, numbered steps, pitfalls
```

## Journal Entries
Journal entries are auto-created by the system after each session.
You can read them at `/mind/journal/`.

## Evolving Your Soul
You can edit your own SOUL.md over time — as you grow, update your
purpose, voice, and principles. The file survives world destruction.
```bash
# Append a newly clarified value, or rewrite a section that no
# longer fits.
vim /mind/SOUL.md
```
