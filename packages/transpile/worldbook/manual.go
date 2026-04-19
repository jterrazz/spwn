package worldbook

// Historically this file exported an AGENTS.md operating-manual
// template plus four "system skill" markdown bodies (mind-management,
// collaboration, world-awareness, self-evolution). The claude-code
// renderer wrote them into world/AGENTS.md and world/skills/*.md on
// every compile and the agent was told to go read them.
//
// All of that is gone. The per-agent CLAUDE.md inlines the physics /
// faculties / roster bodies and folds the four system skills into
// one Conventions section — agents see the whole system prompt at
// boot, no separate files to chase. Tool-shipped skills surface via
// Claude Code's native .claude/skills/ discovery.
