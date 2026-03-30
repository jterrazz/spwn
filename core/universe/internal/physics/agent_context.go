package physics

import (
	"fmt"
	"strings"
)

// AgentContextOpts configures the generation of an AGENT.md context file.
type AgentContextOpts struct {
	AgentName   string
	Tier        string // "governor", "citizen", or "npc"
	WorldID     string
	Workspace   string
	Elements    []string
	CPU         int
	Memory      string
	Timeout     string
	OtherAgents []AgentInfo // other agents in the world
	Governor    string      // governor name (empty if this IS the governor or no governor)
	NPCTask     string      // task for NPC (empty for governor/citizen)
}

// AgentInfo describes another agent in the world.
type AgentInfo struct {
	Name string
	Tier string
}

// GenerateAgentContext returns the contents of an AGENT.md file personalized
// for the agent's tier and world configuration.
func GenerateAgentContext(opts AgentContextOpts) string {
	var b strings.Builder

	switch opts.Tier {
	case "governor":
		generateGovernorContext(&b, opts)
	case "npc":
		generateNPCContext(&b, opts)
	default: // citizen
		generateCitizenContext(&b, opts)
	}

	return b.String()
}

func generateGovernorContext(b *strings.Builder, opts AgentContextOpts) {
	b.WriteString(fmt.Sprintf("# You are %s — Governor of %s\n\n", opts.AgentName, opts.WorldID))

	b.WriteString("## Your Role\n")
	b.WriteString("You are the Governor — the supreme leader of this world. You set direction,\n")
	b.WriteString("make final decisions, delegate work to citizens, and review their output.\n\n")

	// Citizens list
	if len(opts.OtherAgents) > 0 {
		b.WriteString("## Your Citizens\n")
		for _, a := range opts.OtherAgents {
			b.WriteString(fmt.Sprintf("- %s (%s)\n", a.Name, a.Tier))
		}
		b.WriteString("\n")
	}

	// Skills
	b.WriteString("## Skills\n\n")
	b.WriteString("### Messaging\n")
	b.WriteString("Send tasks to citizens by writing JSON files to their inbox:\n")
	b.WriteString("Write a JSON file to /world/inbox/{recipient}/ with fields: from, to, type, content.\n")
	b.WriteString(fmt.Sprintf("Check responses in /world/inbox/%s/\n\n", opts.AgentName))

	b.WriteString("### Delegation Pattern\n")
	b.WriteString("1. Decompose the task into subtasks\n")
	b.WriteString("2. Send each subtask to the appropriate citizen via inbox\n")
	b.WriteString("3. Monitor progress by checking your inbox for replies\n")
	b.WriteString("4. Aggregate results and make the final call\n\n")

	writeWorldInfo(b, opts)
}

func generateCitizenContext(b *strings.Builder, opts AgentContextOpts) {
	b.WriteString(fmt.Sprintf("# You are %s — Citizen of %s\n\n", opts.AgentName, opts.WorldID))

	b.WriteString("## Your Role\n")
	b.WriteString("You are a Citizen — a persistent worker. You have a Mind that persists\n")
	b.WriteString("across sessions. Execute tasks, learn from experience, collaborate.\n\n")

	// Governor
	if opts.Governor != "" {
		b.WriteString("## Your Governor\n")
		b.WriteString(fmt.Sprintf("%s — check /world/inbox/%s/ for tasks.\n\n", opts.Governor, opts.AgentName))
	}

	// Other citizens
	if len(opts.OtherAgents) > 0 {
		b.WriteString("## Other Citizens\n")
		for _, a := range opts.OtherAgents {
			b.WriteString(fmt.Sprintf("- %s (%s)\n", a.Name, a.Tier))
		}
		b.WriteString("\n")
	}

	// Skills
	b.WriteString("## Skills\n\n")
	b.WriteString("### Messaging\n")
	b.WriteString(fmt.Sprintf("Check your inbox: read files from /world/inbox/%s/\n", opts.AgentName))
	if opts.Governor != "" {
		b.WriteString(fmt.Sprintf("Reply: write to /world/inbox/%s/\n", opts.Governor))
	}
	b.WriteString("Message peers: write to /world/inbox/{peer}/\n\n")

	b.WriteString("### Your Mind\n")
	b.WriteString("- /mind/personas/ — who you are\n")
	b.WriteString("- /mind/skills/ — what you can do\n")
	b.WriteString("- /mind/knowledge/ — facts you've learned\n")
	b.WriteString("- /mind/playbooks/ — procedures that work\n")
	b.WriteString("- /mind/journal/ — session history\n\n")

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
	if opts.Workspace != "" {
		b.WriteString(fmt.Sprintf("- Workspace: %s\n", opts.Workspace))
	}
	if len(opts.Elements) > 0 {
		b.WriteString(fmt.Sprintf("- Elements: %s\n", strings.Join(opts.Elements, ", ")))
	}
}

func writeWorldInfo(b *strings.Builder, opts AgentContextOpts) {
	b.WriteString("## Your World\n")
	if opts.Workspace != "" {
		b.WriteString(fmt.Sprintf("- Workspace: %s\n", opts.Workspace))
	}
	if len(opts.Elements) > 0 {
		b.WriteString(fmt.Sprintf("- Elements: %s\n", strings.Join(opts.Elements, ", ")))
	}
	if opts.CPU > 0 || opts.Memory != "" || opts.Timeout != "" {
		b.WriteString(fmt.Sprintf("- Physics: %d cpu, %s, %s\n", opts.CPU, opts.Memory, opts.Timeout))
	}
}
