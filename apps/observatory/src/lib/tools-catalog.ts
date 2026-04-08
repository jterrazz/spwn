/**
 * Static tool catalog — mirrors core/imagebuilder/catalog.
 * Skill content is inlined from the embedded markdown files.
 */

export type ToolKind = "sdk" | "runtime" | "tool" | "platform";
export type ToolStatus = "available" | "planned";

export interface SkillFile {
  name: string; // e.g. "SKILL.md", "fleet-ops.md"
  content: string;
}

export interface ToolDef {
  name: string;
  kind: ToolKind;
  description: string;
  provides: string;
  useWhen: string;
  deps: string[];
  verify: string[];
  status: ToolStatus;
  skills: SkillFile[];
}

export const TOOLS: ToolDef[] = [
  // ── SDKs ──
  {
    name: "@spwn/unix",
    kind: "sdk",
    description: "Core Unix utilities",
    provides: "bash, coreutils, grep, sed, awk, curl, jq",
    useWhen: "You need standard shell tools",
    deps: [],
    verify: ["bash", "grep", "sed", "awk", "curl", "jq"],
    status: "available",
    skills: [],
  },
  {
    name: "@spwn/node",
    kind: "sdk",
    description: "Node.js 20 SDK",
    provides: "node, npm, npx",
    useWhen: "Your project uses JavaScript or TypeScript",
    deps: [],
    verify: ["node", "npm", "npx"],
    status: "available",
    skills: [],
  },
  {
    name: "@spwn/python",
    kind: "sdk",
    description: "Python 3 SDK",
    provides: "python3, pip",
    useWhen: "Your project uses Python",
    deps: [],
    verify: ["python3", "pip3"],
    status: "available",
    skills: [],
  },
  {
    name: "@spwn/build",
    kind: "sdk",
    description: "C/C++ build essentials",
    provides: "make, gcc, g++",
    useWhen: "You need to compile native code",
    deps: [],
    verify: ["make", "gcc", "g++"],
    status: "available",
    skills: [],
  },
  // ── Runtimes ──
  {
    name: "@spwn/claude-code",
    kind: "runtime",
    description: "Claude Code AI runtime",
    provides: "claude CLI + pre-configured auth",
    useWhen: "You want Anthropic's agent runtime (default)",
    deps: ["@spwn/node"],
    verify: ["claude"],
    status: "available",
    skills: [
      {
        name: "SKILL.md",
        content: `# Claude Code

Claude Code is your AI agent runtime — the thinking engine that processes tasks.

## Configuration
Claude Code is pre-configured with:
- Onboarding completed
- Workspace trust granted for /workspace and /home/spwn
- Dangerous mode permissions skipped

## Usage
\`\`\`bash
claude "your task here"              # Run a task
claude --continue                    # Continue last session
claude --session-id <id> "task"      # Resume specific session
\`\`\`

## Environment
- \`ANTHROPIC_API_KEY\` — API key for Claude (if using API auth)
- \`CLAUDE_CODE_OAUTH_TOKEN\` — OAuth token (if using subscription auth)`,
      },
    ],
  },
  {
    name: "@spwn/codex",
    kind: "runtime",
    description: "OpenAI Codex agent runtime",
    provides: "codex CLI + pre-configured workspace trust",
    useWhen: "You want to use OpenAI models (GPT-5, o3) as the agent runtime",
    deps: ["@spwn/node"],
    verify: ["codex"],
    status: "available",
    skills: [
      {
        name: "SKILL.md",
        content: `# Codex

Codex is OpenAI's agent runtime — a CLI that executes tasks using GPT models.

## Usage
\`\`\`bash
codex "your task here"                      # Interactive mode
codex exec "your task here"                 # Non-interactive mode
codex exec "task" --full-auto               # Full auto with sandboxed writes
codex exec "task" --model gpt-5.4           # Specify model
\`\`\`

## Configuration
Codex config lives at \`~/.codex/config.toml\` inside the container.
Auth tokens are forwarded from the host automatically.

## Environment
- Auth is handled via OAuth tokens (subscription-based, e.g. ChatGPT Plus)
- Tokens are mounted from the host at \`~/.codex/auth.json\``,
      },
    ],
  },
  {
    name: "@spwn/aider",
    kind: "runtime",
    description: "Aider code assistant",
    provides: "aider CLI",
    useWhen: "You want an open-source code-focused runtime",
    deps: ["@spwn/python"],
    verify: ["aider"],
    status: "planned",
    skills: [],
  },
  // ── Tools ──
  {
    name: "@spwn/git",
    kind: "tool",
    description: "Git version control",
    provides: "git",
    useWhen: "You need source control (almost always)",
    deps: [],
    verify: ["git"],
    status: "available",
    skills: [],
  },
  {
    name: "@spwn/docker-cli",
    kind: "tool",
    description: "Docker CLI for DooD",
    provides: "docker",
    useWhen: "The agent needs to manage containers",
    deps: [],
    verify: ["docker"],
    status: "available",
    skills: [],
  },
  {
    name: "@spwn/qmd",
    kind: "tool",
    description: "On-device markdown search",
    provides: "qmd — BM25 + semantic search",
    useWhen: "The agent needs to search docs or knowledge bases locally",
    deps: ["@spwn/node"],
    verify: ["qmd"],
    status: "available",
    skills: [
      {
        name: "SKILL.md",
        content: `# QMD — Query Markdown Documents

QMD is an on-device search engine for markdown notes, meeting transcripts, documentation, and knowledge bases.

## Usage
\`\`\`bash
qmd search "your query"              # Search indexed documents
qmd index /path/to/docs              # Index a directory
qmd search --semantic "concept"      # Semantic search with embeddings
\`\`\`

## Features
- BM25 full-text search for keyword matching
- Vector semantic search using local embeddings
- LLM re-ranking for contextual retrieval
- Hybrid approach ideal for agentic workflows

## When to Use
Use QMD when you need to search through large knowledge bases, documentation, or notes.
It runs entirely locally — no external API calls needed for search.`,
      },
    ],
  },
  // ── Platform ──
  {
    name: "@spwn/cli",
    kind: "platform",
    description: "spwn CLI",
    provides: "spwn — agent management, messaging, identity",
    useWhen: "The agent needs to manage its own identity or sub-worlds",
    deps: [],
    verify: ["spwn"],
    status: "available",
    skills: [
      {
        name: "SKILL.md",
        content: `# spwn CLI

The spwn CLI manages worlds, agents, and the universe from inside a container.

## Key Commands
\`\`\`bash
spwn status                        # System status
spwn ls                            # List worlds
spwn agent ls                      # List agents
spwn msg inbox <name>              # Check messages
spwn msg send <to> --from <me> "msg"  # Send message
\`\`\`

## Agent Identity
Your mind is at \`/mind/\` — read your purpose, traits, and persona before starting work.
\`\`\`bash
cat /mind/core/purpose.md
cat /mind/core/persona.md
cat /mind/core/traits.md
\`\`\``,
      },
    ],
  },
  {
    name: "@spwn/architect",
    kind: "platform",
    description: "Orchestration daemon",
    provides: "spwn + claude + docker (full stack)",
    useWhen: "You're running the always-on Architect",
    deps: ["@spwn/cli", "@spwn/claude-code", "@spwn/docker-cli"],
    verify: ["spwn", "claude", "docker"],
    status: "available",
    skills: [
      {
        name: "SKILL.md",
        content: `# Architect

You are the Architect — the always-on daemon that builds and oversees worlds.

## First Things First
1. Read your stack at /me/stack.md — prioritize focus tasks
2. Check system status: \`spwn status\`
3. Address the highest priority task in Focus

## Knowledge Management
You maintain the project knowledge at /universe/knowledge/.
EVERY conversation should result in knowledge updates.`,
      },
      {
        name: "fleet-ops.md",
        content: `# Fleet Operations

## Managing Worlds
\`\`\`bash
spwn ls                           # List all worlds
spwn up --agent <name> -w <path>  # Spawn a world
spwn down <id>                    # Destroy a world
spwn inspect <id>                 # World details
\`\`\`

## Managing Agents
\`\`\`bash
spwn agent ls                     # List all agents
spwn agent new <name>             # Create agent
spwn agent rm <name>              # Remove agent
spwn agent talk <name> "msg"      # Talk to agent
spwn profile <name>               # View profile
\`\`\`

## Agent Lifecycle
1. Create: \`spwn agent new <name>\`
2. Configure: write purpose, persona, traits
3. Spawn: \`spwn up --agent <name> -w <workspace>\`
4. Work: \`spwn agent talk <name> "task"\`
5. Dream: \`spwn agent dream <name>\` (promote patterns)
6. Sleep: \`spwn agent sleep <name>\` (consolidate)`,
      },
      {
        name: "monitoring.md",
        content: `# Monitoring

## Health Checks
\`\`\`bash
spwn status                       # Overall system status
spwn ls                           # Running worlds
spwn agent ls                     # All agents
\`\`\`

## Agent Health
Check an agent's journal for recent activity:
\`\`\`bash
spwn profile <name> journal       # View journal entries
spwn profile <name> knowledge     # View knowledge files
\`\`\`

## Responding to Issues
- World crashed: check logs, respawn
- Agent idle: send a message or restart
- Memory full: trigger sleep cycle`,
      },
      {
        name: "task-planning.md",
        content: `# Task Planning

## Structured Response Format
When managing tasks, use these markers:

### Pushing a task
\`\`\`
[STACK_PUSH] Short task title
Priority: blocking|queued
Brief description of what you'll do.
\`\`\`

### Popping a task (completing)
\`\`\`
[STACK_POP] Short task title
Done: brief summary of what was done.
\`\`\`

## Planning Workflow
1. Read stack at start of every interaction
2. Prioritize: what's most impactful?
3. Break large tasks into sub-tasks
4. Assign to agents or do yourself
5. Update stack after completing work`,
      },
    ],
  },
];

/** Lookup a tool by name. */
export function getToolByName(name: string): ToolDef | undefined {
  return TOOLS.find((t) => t.name === name);
}

/** Get the slug from a tool name (e.g. "@spwn/qmd" → "qmd"). */
export function toolSlug(name: string): string {
  return name.replace("@spwn/", "");
}

/** Kind display info. */
export const KIND_META: Record<ToolKind, { label: string; color: string }> = {
  sdk: { label: "SDK", color: "bg-blue-500/15 text-blue-400/80 border-blue-500/20" },
  runtime: { label: "Runtime", color: "bg-purple-500/15 text-purple-400/80 border-purple-500/20" },
  tool: { label: "Tool", color: "bg-green-500/15 text-green-400/80 border-green-500/20" },
  platform: { label: "Platform", color: "bg-amber-500/15 text-amber-400/80 border-amber-500/20" },
};
