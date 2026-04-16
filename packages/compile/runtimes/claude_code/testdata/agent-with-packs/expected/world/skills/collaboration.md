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

## Understanding Your Role
- **Leader**: You orchestrate other agents. You delegate tasks and coordinate work.
- **Worker**: You do focused work. You report to your leader.
- **Ephemeral**: You execute a single task and exit.

## Working with Your Leader
If you have a leader, check `/world/AGENT.md` for instructions.
Report progress by writing to your journal.
