# Mind Management

## Reading Your Identity
Before starting any task, read your identity files:
```bash
cat /mind/core/purpose.md   # Why you exist
cat /mind/core/profile.md   # Who you are
cat /mind/core/traits.md    # Your principles
```

## Saving Knowledge
When you discover something worth remembering:
```bash
# Create a knowledge file with a descriptive name
echo "# What I learned about X" > /mind/knowledge/topic-name.md
```
Keep files focused on ONE topic. Use clear filenames.

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
echo "# Purpose\nI exist to maintain the production API" > /mind/core/purpose.md
```
