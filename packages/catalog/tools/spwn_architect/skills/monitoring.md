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
spwn profile <name> journal       # View journal entries
spwn profile <name> knowledge     # View knowledge files
```

## Responding to Issues
- World crashed: check logs, respawn
- Agent idle: send a message or restart
- Memory full: trigger sleep cycle
