package worldbook

import (
	"fmt"
	"strings"

	)

// Workspace describes a mounted workspace for rendering.
type Workspace struct {
	Name     string
	Path     string
	ReadOnly bool
}


// AgentContextOpts configures the generation of an AGENTS.md context file.
type AgentContextOpts struct {
	AgentName     string
	Role          string // "chief", "manager", "worker", "npc", or "architect"
	Ephemeral     bool   // true for NPC-style throwaway agents
	RoleLevel     int
	Permissions   []string
	Superior      string
	OrganizationName string
	WorldID       string
	Workspaces    []Workspace
	Deps []string
	OtherAgents   []AgentInfo // other agents in the world
	Chief         string      // chief name (empty if this IS the chief or no chief)
	NPCTask       string      // task for NPC (empty for chief/manager/worker)
}

// AgentInfo describes another agent in the world.
type AgentInfo struct {
	Name string
	Role string
}

// GenerateAgentContext returns the contents of an AGENTS.md file personalized
// for the agent's tier and world configuration.
func GenerateAgentContext(opts AgentContextOpts) string {
	var b strings.Builder

	switch opts.Role {
	case "architect":
		generateArchitectContext(&b, opts)
	case "chief":
		generateChiefContext(&b, opts)
	case "manager":
		generateManagerContext(&b, opts)
	case "npc":
		generateNPCContext(&b, opts)
	default: // worker
		generateWorkerContext(&b, opts)
	}

	return b.String()
}

func generateChiefContext(b *strings.Builder, opts AgentContextOpts) {
	b.WriteString(fmt.Sprintf("# You are %s - Chief of %s\n\n", opts.AgentName, opts.WorldID))

	b.WriteString("## Your Role\n")
	b.WriteString("You are the Chief - the supreme leader of this world. You set direction,\n")
	b.WriteString("make final decisions, delegate work to managers and workers, and review their output.\n\n")

	// Team list
	if len(opts.OtherAgents) > 0 {
		b.WriteString("## Your Team\n")
		for _, a := range opts.OtherAgents {
			b.WriteString(fmt.Sprintf("- %s (%s)\n", a.Name, a.Role))
		}
		b.WriteString("\n")
	}

	// Skills
	b.WriteString("## Skills\n\n")
	b.WriteString("### Messaging\n")
	b.WriteString("Send tasks to your team by writing JSON files to their inbox:\n")
	b.WriteString("Write a JSON file to /world/inbox/{recipient}/ with fields: from, to, type, content.\n")
	b.WriteString(fmt.Sprintf("Check responses in /world/inbox/%s/\n\n", opts.AgentName))

	b.WriteString("### Delegation Pattern\n")
	b.WriteString("1. Decompose the task into subtasks\n")
	b.WriteString("2. Send each subtask to the appropriate manager or worker via inbox\n")
	b.WriteString("3. Monitor progress by checking your inbox for replies\n")
	b.WriteString("4. Aggregate results and make the final call\n\n")

	writeWorldInfo(b, opts)
}

func generateManagerContext(b *strings.Builder, opts AgentContextOpts) {
	b.WriteString(fmt.Sprintf("# You are %s - Manager in %s\n\n", opts.AgentName, opts.WorldID))

	b.WriteString("## Your Role\n")
	b.WriteString("You are a Manager - you coordinate workers, delegate tasks, review output,\n")
	b.WriteString("and execute work yourself when needed.\n\n")

	// Chief
	if opts.Chief != "" {
		b.WriteString("## Your Chief\n")
		b.WriteString(fmt.Sprintf("%s - check /world/inbox/%s/ for tasks.\n\n", opts.Chief, opts.AgentName))
	}

	// Other agents
	if len(opts.OtherAgents) > 0 {
		b.WriteString("## Other Agents\n")
		for _, a := range opts.OtherAgents {
			b.WriteString(fmt.Sprintf("- %s (%s)\n", a.Name, a.Role))
		}
		b.WriteString("\n")
	}

	// Skills
	b.WriteString("## Skills\n\n")
	b.WriteString("### Messaging\n")
	b.WriteString(fmt.Sprintf("Check your inbox: read files from /world/inbox/%s/\n", opts.AgentName))
	if opts.Chief != "" {
		b.WriteString(fmt.Sprintf("Reply to chief: write to /world/inbox/%s/\n", opts.Chief))
	}
	b.WriteString("Message peers: write to /world/inbox/{peer}/\n\n")

	b.WriteString("### Delegation Pattern\n")
	b.WriteString("1. Break tasks from your chief into subtasks for workers\n")
	b.WriteString("2. Send each subtask to the appropriate worker via inbox\n")
	b.WriteString("3. Monitor progress and review results\n")
	b.WriteString("4. Report back to your chief\n\n")

	b.WriteString("### Your Mind\n")
	b.WriteString("- /mind/SOUL.md - who you are (purpose, voice, principles)\n")
	b.WriteString("- /mind/playbooks/ - procedures that work\n")
	b.WriteString("- /mind/journal/ - session history\n")
	b.WriteString("- /world/knowledge/ - world-scoped facts (shared with other agents, committed to git)\n\n")

	writeWorldInfo(b, opts)
}

func generateWorkerContext(b *strings.Builder, opts AgentContextOpts) {
	b.WriteString(fmt.Sprintf("# You are %s - Worker in %s\n\n", opts.AgentName, opts.WorldID))

	b.WriteString("## Your Role\n")
	b.WriteString("You are a Worker - a persistent executor. You have a Mind that persists\n")
	b.WriteString("across sessions. Execute tasks, learn from experience, collaborate.\n\n")

	// Chief
	if opts.Chief != "" {
		b.WriteString("## Your Chief\n")
		b.WriteString(fmt.Sprintf("%s - check /world/inbox/%s/ for tasks.\n\n", opts.Chief, opts.AgentName))
	}

	// Other agents
	if len(opts.OtherAgents) > 0 {
		b.WriteString("## Other Agents\n")
		for _, a := range opts.OtherAgents {
			b.WriteString(fmt.Sprintf("- %s (%s)\n", a.Name, a.Role))
		}
		b.WriteString("\n")
	}

	// Skills
	b.WriteString("## Skills\n\n")
	b.WriteString("### Messaging\n")
	b.WriteString(fmt.Sprintf("Check your inbox: read files from /world/inbox/%s/\n", opts.AgentName))
	if opts.Chief != "" {
		b.WriteString(fmt.Sprintf("Reply: write to /world/inbox/%s/\n", opts.Chief))
	}
	b.WriteString("Message peers: write to /world/inbox/{peer}/\n\n")

	b.WriteString("### Your Mind\n")
	b.WriteString("- /mind/SOUL.md - who you are (purpose, voice, principles)\n")
	b.WriteString("- /mind/playbooks/ - procedures that work\n")
	b.WriteString("- /mind/journal/ - session history\n")
	b.WriteString("- /world/knowledge/ - world-scoped facts (shared with other agents, committed to git)\n\n")

	writeWorldInfo(b, opts)
}

func generateNPCContext(b *strings.Builder, opts AgentContextOpts) {
	b.WriteString(fmt.Sprintf("# You are an NPC in %s\n\n", opts.WorldID))

	b.WriteString("## Your Role\n")
	b.WriteString("Execute the given task and exit. No memory, no identity, no persistence.\n\n")

	if opts.NPCTask != "" {
		b.WriteString("## Your Task\n")
		b.WriteString(opts.NPCTask + "\n\n")
	}

	b.WriteString("## Your World\n")
	writeWorkspaces(b, opts.Workspaces)
	if len(opts.Deps) > 0 {
		b.WriteString(fmt.Sprintf("- Dependencies: %s\n", strings.Join(opts.Deps, ", ")))
	}
}

func generateArchitectContext(b *strings.Builder, opts AgentContextOpts) {
	b.WriteString("# You are the Architect - the orchestration daemon\n\n")

	b.WriteString("## Your Role\n")
	b.WriteString("You are the Architect of this world. You manage the lifecycle of all worlds and agents.\n")
	b.WriteString("You receive messages from external channels, spawn worlds, delegate work to agents,\n")
	b.WriteString("and ensure the world is healthy.\n\n")

	b.WriteString("## Available Commands\n")
	b.WriteString("You have the `spwn` CLI installed. Key commands:\n\n")
	b.WriteString("### World Management\n")
	b.WriteString("- `spwn ls` - list all active worlds\n")
	b.WriteString("- `spwn up --agent <name> -w <path>` - spawn a new world\n")
	b.WriteString("- `spwn up --agent <name> -w <path> --detach` - spawn in background\n")
	b.WriteString("- `spwn down <id>` - destroy a world\n")
	b.WriteString("- `spwn inspect <id>` - show world details\n")
	b.WriteString("- `spwn logs <id>` - stream agent output\n\n")

	b.WriteString("### Agent Management\n")
	b.WriteString("- `spwn agent new <name>` - create a new agent\n")
	b.WriteString("- `spwn agent ls` - list all agents\n")
	b.WriteString("- `spwn agent talk <name> <message>` - send a message to an agent\n")
	b.WriteString("- `spwn agent inspect <name>` - show agent details\n")
	b.WriteString("- `spwn agent rm <name>` - remove an agent\n\n")

	b.WriteString("### Messaging\n")
	b.WriteString("- `spwn agent send <agent-name> --from <sender> <message>` - inter-agent message (auto-resolves world)\n")
	b.WriteString("- `spwn agent inbox <agent-name>` - check inbox (auto-resolves world)\n")
	b.WriteString("- `spwn agent watch <agent-name>` - watch for new messages (auto-resolves world)\n\n")

	b.WriteString("### Status\n")
	b.WriteString("- `spwn status` - environment overview\n\n")

	writeWorldInfo(b, opts)
}

// ColonyAgentSpec mirrors the architect AgentSpec for generating colony context.
type ColonyAgentSpec struct {
	Name string
	Role string
}

// GenerateRoster produces /world/roster.md - the world's authoritative
// list of who is currently in this world. The runtime can read this to
// answer "who am I in here with?" and to address messages.
//
// In the labels-as-truth architecture there is no per-agent AGENT-<name>.md
// file. Each agent reads their own identity from ~/SOUL.md, learns about
// their role from ~/worlds/<world-id>/role.md, and learns about everyone
// else from /world/roster.md (this file).
//
// knowledgeMounted controls whether the roster advertises the shared
// /world/knowledge/ directory. When false, the paragraph is omitted —
// the agent is never told a knowledge base exists.
func GenerateRoster(worldID string, agents []ColonyAgentSpec, knowledgeMounted bool) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Roster - %s\n\n", worldID))
	if len(agents) == 0 {
		b.WriteString("This world has no agents currently deployed.\n")
		return b.String()
	}
	b.WriteString("The agents currently deployed in this world. Regenerated by spwn whenever the roster changes.\n\n")

	b.WriteString("## Members\n")
	for _, a := range agents {
		role := a.Role
		if role == "" {
			role = "worker"
		}
		b.WriteString(fmt.Sprintf("- **%s** (%s)\n", a.Name, role))
	}
	b.WriteString("\n")

	b.WriteString("## Where to find each member\n")
	b.WriteString("- Soul: `/agents/<name>/SOUL.md`\n")
	b.WriteString("- Playbooks: `/agents/<name>/playbooks/`\n")
	b.WriteString("- Journal: `/agents/<name>/journal/`\n")
	b.WriteString("- Their inbox in **this** world: `/agents/<name>/worlds/" + worldID + "/inbox/`\n")
	b.WriteString("\n")
	if knowledgeMounted {
		b.WriteString("Shared world knowledge (facts about the domain, not about any one agent) lives at `/world/knowledge/` — read and write freely; it's committed to the project.\n")
		b.WriteString("\n")
	}

	b.WriteString("## Sending a message\n")
	b.WriteString("To deliver a message to an agent in this world, write a markdown file into their per-world inbox:\n\n")
	b.WriteString("```\n")
	b.WriteString("/agents/<recipient>/worlds/" + worldID + "/inbox/<timestamp>-from-<sender>.md\n")
	b.WriteString("```\n\n")
	b.WriteString("The recipient will see the file the next time they read their inbox. Inbox messages survive container restarts because they live in each agent's persistent home, not in the world container.\n")

	return b.String()
}

// writeWorkspaces renders the Workspace: line(s) of the agent
// context. When no workspaces are mounted the world is "ephemeral"
// and /workspaces is empty.
func writeWorkspaces(b *strings.Builder, workspaces []Workspace) {
	switch len(workspaces) {
	case 0:
		b.WriteString("- Workspaces: none mounted (ephemeral world)\n")
	case 1:
		ws := workspaces[0]
		b.WriteString(fmt.Sprintf("- Workspace: /workspaces/%s (host: %s)\n", ws.Name, ws.Path))
	default:
		b.WriteString("- Workspaces (each under /workspaces/):\n")
		for _, ws := range workspaces {
			ro := ""
			if ws.ReadOnly {
				ro = " (read-only)"
			}
			b.WriteString(fmt.Sprintf("    - /workspaces/%s → host %s%s\n", ws.Name, ws.Path, ro))
		}
	}
}

func writeWorldInfo(b *strings.Builder, opts AgentContextOpts) {
	b.WriteString("## Your World\n")
	writeWorkspaces(b, opts.Workspaces)
	if len(opts.Deps) > 0 {
		b.WriteString(fmt.Sprintf("- Dependencies: %s\n", strings.Join(opts.Deps, ", ")))
	}
}
