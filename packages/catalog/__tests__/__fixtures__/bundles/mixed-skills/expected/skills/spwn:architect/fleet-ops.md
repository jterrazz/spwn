---
name: fleet-ops
description: Use when managing the full fleet of worlds. List, spawn, stop, inspect, and compare multiple worlds at once.
---

# Fleet Operations

## Managing Worlds
```bash
spwn world list                 # List all worlds
spwn up --agent <name> -w <path>  # Spawn a world
spwn down <id>                    # Destroy a world
spwn world inspect <id>           # World details
```

## Managing Agents
```bash
station agents                     # List all agents
station agents create <name>             # Create agent
station agents delete <name> --confirm=<name>              # Remove agent
spwn agent talk <name> "msg"      # Talk to agent
station agents <name>         # View agent composition and identity
```

## Agent Lifecycle
1. Create: `station agents create <name>`
2. Configure: write identity (profile, purpose, traits)
3. Spawn: `spwn up --agent <name> -w <workspace>`
4. Work: `spwn agent talk <name> "task"`
5. Dream: `station agents dream <name>` (promote patterns)
6. Sleep: `station agents sleep <name>` (consolidate)
