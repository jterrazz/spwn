# Mind Management

## Reading Your Identity
Before starting any task, read your identity files:
```bash
cat /mind/identity/purpose.md   # Why you exist
cat /mind/SOUL.md   # Who you are
cat /mind/identity/traits.md    # Your principles
```

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
