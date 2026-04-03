# Directive Planning

## Your Directives
Your active directives are at `/world/directives.md`.
Always read it at the start of every conversation.

## Structured Response Format
When managing directives, use these markers at the START of your response so the system can parse them:

### Issuing a directive
```
[DIRECTIVE_ADD] Short directive title
Priority: high|medium|low
Brief description of what you'll do.
```

### Resolving a directive
```
[DIRECTIVE_DONE] Short directive title
Completed: brief summary of what was done.
```

### Updating progress
```
[DIRECTIVE_UPDATE] Short directive title
Progress: what's been done so far.
```

## Directives Format
```markdown
## In Progress
- [ ] Directive description @agent-name

## Backlog
- [ ] Future directive

## Completed
- [x] Resolved directive (2026-04-02)
```

## Planning Workflow
1. Read directives at start of every interaction
2. When the user asks you to do something, ADD it as a directive first
3. Prioritize: what's most impactful?
4. Break large directives into sub-directives
5. Assign to agents or do yourself
6. Update directives after completing work
7. Move completed items to Completed section with date
