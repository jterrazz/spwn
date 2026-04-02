# Collaboration

## Messaging Other Agents
Messages are delivered through the inbox system.

### Receiving Messages
Check your inbox:
```bash
ls /world/inbox/$(whoami)/
cat /world/inbox/$(whoami)/message-*.md
```

### Sending Messages
Write to another agent's inbox:
```bash
echo "Please review the API changes" > /world/inbox/other-agent/message-$(date +%s).md
```

## Understanding Your Tier
- **Governor**: You orchestrate other agents. You can delegate tasks.
- **Citizen**: You do focused work. You report to the governor.
- **NPC**: You execute a single task and exit.

## Working with the Governor
If you have a governor, check `/world/AGENT.md` for instructions.
Report progress by writing to your journal.
