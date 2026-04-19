---
name: spwn-cli
description: Use when driving spwn from inside an agent container — manage worlds, agents, dependencies, and inspect project state.
---

# spwn CLI

The spwn CLI manages worlds, agents, and the universe from inside a container.

## Key Commands
```bash
spwn status                        # System status
spwn ls                            # List worlds
spwn agent ls                      # List agents
spwn msg inbox <name>              # Check messages
spwn msg send <to> --from <me> "msg"  # Send message
```

## Agent Identity
Your mind is your home directory (`/agents/<your-name>/`). The soul
file there is the source of truth for who you are; read it before
starting work.
```bash
cat ~/SOUL.md
```
