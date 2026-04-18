---
name: world-exploration
description: Use when first waking up in a new spwn world to systematically map what's installed, what's mounted, and what the agent can touch.
---

# Skill: World Exploration

How to systematically explore a spwn world when you first wake up.

## The standard tour

Every spwn world has a predictable layout. Walk it in this order:

### 1. /world/ — The world manifest

```
ls /world/
cat /world/physics.md    # network, resource limits, runtime rules
cat /world/faculties.md  # what tools and deps are available
```

This tells you what kind of environment you are in: how much memory,
what network access exists, which languages are installed.

### 2. ~/identity/ — Who you are

```
cat ~/SOUL.md
```

Your persona, voice, and purpose. Re-read this if you ever feel lost.

### 3. /workspaces/ — Mounted projects

```
ls /workspaces/
```

If a workspace is mounted, explore its structure before modifying anything:

```
find /workspaces/ -maxdepth 2 -type f | head -30
cat /workspaces/*/README.md 2>/dev/null
```

### 4. Toolchain check

Verify what is actually installed vs what the manifest claims:

```
which git && git --version
which node && node --version
which python3 && python3 --version
```

Report any gaps between the manifest and reality.

## Exploration principles

- **Read before you write.** Understand the filesystem before creating files.
- **Go shallow first.** List directories before diving into files.
- **Report what you find.** Every discovery should be narrated to the user.
- **Note surprises.** If something is missing or unexpected, say so immediately.
