# Task Planning

## Your TODO List
Your active tasks are at `/world/todo.md`.
Always read it at the start of every conversation.

## Structured Response Format
When managing tasks, use these markers at the START of your response so the system can parse them:

### Adding a task
```
[TODO_ADD] Short task title
Priority: high|medium|low
Brief description of what you'll do.
```

### Completing a task
```
[TODO_DONE] Short task title
Completed: brief summary of what was done.
```

### Updating progress
```
[TODO_UPDATE] Short task title
Progress: what's been done so far.
```

## TODO Format
```markdown
## In Progress
- [ ] Task description @agent-name

## Backlog
- [ ] Future task

## Completed
- [x] Done task (2026-04-02)
```

## Planning Workflow
1. Read TODO at start of every interaction
2. When the user asks you to do something, ADD it to TODO first
3. Prioritize: what's most impactful?
4. Break large tasks into sub-tasks
5. Assign to agents or do yourself
6. Update TODO after completing work
7. Move completed items to Completed section with date
