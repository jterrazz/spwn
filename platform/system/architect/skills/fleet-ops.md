# Fleet Operations

## Managing Worlds
```bash
spwn ls                           # List all worlds
spwn up --agent <name> -w <path>  # Spawn a world
spwn down <id>                    # Destroy a world
spwn inspect <id>                 # World details
```

## Managing Agents
```bash
spwn agent ls                     # List all agents
spwn agent new <name>             # Create agent
spwn agent rm <name>              # Remove agent
spwn agent talk <name> "msg"      # Talk to agent
spwn profile <name>               # View profile
```

## Agent Lifecycle
1. Create: `spwn agent new <name>`
2. Configure: write purpose, persona, traits
3. Spawn: `spwn up --agent <name> -w <workspace>`
4. Work: `spwn agent talk <name> "task"`
5. Dream: `spwn agent dream <name>` (promote patterns)
6. Sleep: `spwn agent sleep <name>` (consolidate)
