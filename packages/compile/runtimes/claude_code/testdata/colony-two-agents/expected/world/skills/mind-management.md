# Mind Management

## Reading Your Identity
Before starting any task, read your identity files:
```bash
cat /mind/identity/purpose.md   # Why you exist
cat /mind/identity/profile.md   # Who you are
cat /mind/identity/traits.md    # Your principles
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

## Updating Your Identity
You can evolve your own identity:
```bash
# Update your purpose as you learn
echo "# Purpose\nI exist to maintain the production API" > /mind/identity/purpose.md
```
