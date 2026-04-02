# Mind Management

## Reading Your Identity
Before starting any task, read your identity files:
```bash
cat /mind/identity/purpose.md   # Why you exist
cat /mind/identity/persona.md   # Who you are
cat /mind/identity/traits.md    # Your principles
cat /mind/bonds.md              # Your relationships
```

## Saving Knowledge
When you discover something worth remembering:
```bash
# Create a knowledge file with a descriptive name
echo "# What I learned about X" > /mind/memory/knowledge/topic-name.md
```
Keep files focused on ONE topic. Use clear filenames.

## Creating Playbooks
When you find a reusable procedure:
```bash
echo "# How to Deploy" > /mind/memory/playbooks/deploy.md
# Include: trigger conditions, numbered steps, pitfalls
```

## Journal Entries
Journal entries are auto-created by the system after each session.
You can read them at `/mind/memory/journal/`.

## Updating Your Identity
You can evolve your own identity:
```bash
# Update your purpose as you learn
echo "# Purpose\nI exist to maintain the production API" > /mind/identity/purpose.md
```
