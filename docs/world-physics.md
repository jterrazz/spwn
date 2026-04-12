# World Physics

The world manifest defines what is physically possible inside a container.

## Configuration

```yaml
physics:
  constants:
    cpu: 2
    memory: 1GB
    timeout: 30m

tools:
  - @spwn/unix          # bash, coreutils, grep, sed, awk
  - @spwn/git           # version control
  - @spwn/node          # Node.js 20 + npm
  - @spwn/claude-code   # AI agent runtime
  - @spwn/cli           # spwn CLI
  - @spwn/qmd           # on-device markdown search

gate:
  - source: mcp/slack
    as: slack-send
    capabilities: [send]
```

If `curl` is not in the tools list, it does not exist. Tools are composable, dependency-aware, and verified at world creation. Each tool ships its own skills (Vercel SKILL.md convention). The image is built on-demand from your exact tool selection — no bloated base images.

## Security model

No ACLs. No permission prompts. If a binary isn't installed, it's **physically impossible** — not "forbidden," absent. You can't prompt-inject a missing binary.

| Without Spwn | With Spwn |
| --- | --- |
| No structure for managing multiple agents. | Flexible hierarchy with messaging and delegation. |
| Setup isn't reproducible or shareable. | Declarative YAML config. Git-friendly, shareable. |
| Can't see what tools and skills are exposed. | `spwn inspect` shows everything. |
| Zero governance. No cost limits. | `org.yaml` defines governance. |
| Agent forgets everything between sessions. | Persistent identity survives across worlds. |
| One bad agent action compromises your host. | Fully isolated Docker worlds. Snapshots and rollback. |

## Why Spwn is different

| | |
|---|---|
| **Hierarchy over flat pools.** | Architect → Universe → World → Hierarchy roles. Clear structure, clear delegation. |
| **Worlds over wrappers.** | Not another API layer. A full environment with filesystem, compute, memory, and network. |
| **Identity over instances.** | Agents have persistent purpose, traits, skills, and memory. They're individuals, not stateless functions. |
| **Agency over tools.** | MCP gives agents a Swiss Army knife. Spwn gives them a workshop. They discover, compose, and create. |
| **Physics over permissions.** | No ACLs. No allowlists. If curl isn't installed, HTTP is impossible. Security is structural. |
| **Evolution over configuration.** | Agents learn from tasks via dream, consolidate knowledge during sleep, and branch via forking. |
