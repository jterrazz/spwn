# Monitoring

## Health Checks
```bash
spwn status                       # Overall system status
spwn ls                           # Running worlds
spwn agent ls                     # All agents
```

## Agent Health
Check an agent's journal for recent activity:
```bash
spwn agent inspect <name>         # Composition, memory, recent journal
spwn agent logs <name>            # Event log for this agent
```

## Responding to Issues
- World crashed: check logs, respawn
- Agent idle: send a message or restart
- Memory full: trigger sleep cycle
