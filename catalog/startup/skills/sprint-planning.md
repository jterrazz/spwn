---
name: sprint-planning
description: Use when planning and structuring a week of work across the startup colony — priorities, handoffs, definitions of done.
---

# Skill: Sprint Planning

How to plan and structure a week of work across the startup colony.

## The weekly cycle

Every Monday (or when the CEO initiates a planning cycle), follow this process:

### 1. Gather status

```
# Read what each agent accomplished last week
cat /agents/devops/journal/latest.md
cat /agents/analyst/journal/latest.md
cat /agents/ceo/journal/latest.md
```

### 2. Identify priorities

Review the backlog and last week's outcomes. Ask:

- What shipped? What didn't? Why?
- What did the analyst find that changes our direction?
- Is the pipeline healthy enough to ship something new?

### 3. Write the sprint plan

Create a sprint document with:

- **Goal**: one sentence describing what success looks like this week.
- **Tasks**: 2-3 concrete deliverables, each assigned to an agent.
- **Risks**: anything that could block the sprint.

```
# Save to your journal
echo "## Sprint $(date +%Y-%W)" >> /agents/ceo/journal/sprints.md
```

### 4. Communicate

Message each agent with their tasks:

```
# Via inbox
spwn msg send devops "Sprint task: ..."
spwn msg send analyst "Research question: ..."
```

## Planning principles

- **Small batches.** Two things done beats five things started.
- **One goal.** If you can't say the sprint goal in one sentence, it's too big.
- **No surprises.** Every agent should know what's expected before they start.
- **Review last week first.** Don't plan forward without looking back.
