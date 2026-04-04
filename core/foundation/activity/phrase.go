package activity

import (
	"fmt"
	"strings"
)

// Phrase helpers centralize natural-language event descriptions.
// All phrases are authored here so the copy stays consistent.

// extractName pulls the human-readable name out of an ID like "w-saturn-12345".
func extractName(id string) string {
	parts := strings.Split(id, "-")
	if len(parts) >= 2 {
		n := parts[1]
		if len(n) > 0 {
			return strings.ToUpper(n[:1]) + n[1:]
		}
	}
	return id
}

func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	default:
		return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
	}
}

// PhraseWorldSpawned: "Architect spawned Saturn with Neo and Morpheus"
func PhraseWorldSpawned(worldID string, agents []string) string {
	name := extractName(worldID)
	if len(agents) == 0 {
		return fmt.Sprintf("Architect spawned %s", name)
	}
	return fmt.Sprintf("Architect spawned %s with %s", name, joinNames(agents))
}

// PhraseWorldDestroyed: "Saturn was destroyed after 47m"
func PhraseWorldDestroyed(worldID string, uptime string) string {
	name := extractName(worldID)
	if uptime == "" {
		return fmt.Sprintf("%s was destroyed", name)
	}
	return fmt.Sprintf("%s was destroyed after %s", name, uptime)
}

// PhraseAgentJoined: "Neo joined Saturn"
func PhraseAgentJoined(agentName, worldID string) string {
	return fmt.Sprintf("%s joined %s", agentName, extractName(worldID))
}

// PhraseAgentLeft: "Neo left Saturn after 47m"
func PhraseAgentLeft(agentName, worldID string, uptime string) string {
	name := extractName(worldID)
	if uptime == "" {
		return fmt.Sprintf("%s left %s", agentName, name)
	}
	return fmt.Sprintf("%s left %s after %s", agentName, name, uptime)
}

// PhraseAgentCreated: "You created Neo"
func PhraseAgentCreated(agentName string) string {
	return fmt.Sprintf("You created %s", agentName)
}

// PhraseAgentDeleted: "Neo was deleted"
func PhraseAgentDeleted(agentName string) string {
	return fmt.Sprintf("%s was deleted", agentName)
}

// PhraseAgentDreamed: "Neo dreamed and promoted 2 patterns"
func PhraseAgentDreamed(agentName string, promoted int) string {
	if promoted == 0 {
		return fmt.Sprintf("%s dreamed", agentName)
	}
	suffix := "pattern"
	if promoted != 1 {
		suffix = "patterns"
	}
	return fmt.Sprintf("%s dreamed and promoted %d %s", agentName, promoted, suffix)
}

// PhraseAgentSlept: "Neo slept, archiving 4 playbooks"
func PhraseAgentSlept(agentName string, archived int) string {
	if archived == 0 {
		return fmt.Sprintf("%s slept", agentName)
	}
	suffix := "playbook"
	if archived != 1 {
		suffix = "playbooks"
	}
	return fmt.Sprintf("%s slept, archiving %d %s", agentName, archived, suffix)
}

// PhraseAgentForked: "Morpheus forked from Neo"
func PhraseAgentForked(source, target string) string {
	return fmt.Sprintf("%s forked from %s", target, source)
}

// PhraseAgentTalked: "Neo replied in Saturn"
func PhraseAgentTalked(agentName, worldID string) string {
	if worldID == "" {
		return fmt.Sprintf("%s replied", agentName)
	}
	return fmt.Sprintf("%s replied in %s", agentName, extractName(worldID))
}

// PhraseArchitectStarted: "Architect came online"
func PhraseArchitectStarted() string { return "Architect came online" }

// PhraseArchitectStopped: "Architect went offline"
func PhraseArchitectStopped() string { return "Architect went offline" }

// PhraseArchitectTalked: "Architect received an instruction"
func PhraseArchitectTalked() string { return "Architect received an instruction" }

// PhraseSessionEnded: "Neo finished in Saturn after 47m"
func PhraseSessionEnded(agentName, worldID, uptime, outcome string) string {
	world := extractName(worldID)
	verb := "finished"
	if outcome == "failed" {
		verb = "failed"
	}
	if uptime == "" {
		return fmt.Sprintf("%s %s in %s", agentName, verb, world)
	}
	return fmt.Sprintf("%s %s in %s after %s", agentName, verb, world, uptime)
}

// PhraseWorldSnapshot: "Saturn saved as snapshot-12345"
func PhraseWorldSnapshot(worldID, snapshotName string) string {
	return fmt.Sprintf("%s saved as %s", extractName(worldID), snapshotName)
}
