package architect

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"spwn.sh/core/agent"
	"spwn.sh/core/foundation"
	"spwn.sh/core/foundation/activity"
	"spwn.sh/core/universe/internal/models"
)

// formatUptime returns a human-readable duration like "47m" or "2h".
func formatUptime(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d/time.Hour))
	}
	return fmt.Sprintf("%dd", int(d/(24*time.Hour)))
}

// Destroy stops and removes a world.
func (a *Architect) Destroy(ctx context.Context, worldID string) (*models.World, error) {
	u, err := a.state.Get(worldID)
	if err != nil {
		return nil, err
	}

	a.backend.Stop(ctx, u.ContainerID)
	a.backend.Remove(ctx, u.ContainerID)

	// Write journal entries for every agent that was deployed in this
	// world. Each agent's journal lives in their persistent home dir
	// (~/.spwn/agents/<name>/memory/journal/), reachable via the agent
	// name alone — no per-world MindPath needed.
	duration := time.Since(u.CreatedAt)
	agentNamesForJournal := []string{}
	for _, rec := range u.Agents {
		agentNamesForJournal = append(agentNamesForJournal, rec.Name)
	}
	if len(agentNamesForJournal) == 0 && u.Agent != "" {
		agentNamesForJournal = append(agentNamesForJournal, u.Agent)
	}
	for _, name := range agentNamesForJournal {
		agentPath := filepath.Join(foundation.AgentsDir(), name)
		if journalErr := agent.AppendJournal(agentPath, worldID, -1, duration); journalErr != nil {
			log.Printf("warning: failed to write journal for agent %s on destroy: %v", name, journalErr)
		}
	}

	a.state.Delete(worldID)

	// Emit activity events
	uptime := formatUptime(duration)
	agentNames := []string{}
	for _, rec := range u.Agents {
		agentNames = append(agentNames, rec.Name)
	}
	if len(agentNames) == 0 && u.Agent != "" {
		agentNames = append(agentNames, u.Agent)
	}
	for _, name := range agentNames {
		activity.Log(activity.Event{
			Type:       activity.TypeAgentLeft,
			Actor:      "architect",
			Verb:       "left",
			Target:     worldID,
			Phrase:     activity.PhraseAgentLeft(name, worldID, uptime),
			WorldID:    worldID,
			AgentID:    name,
			DurationMs: duration.Milliseconds(),
		})
	}
	activity.Log(activity.Event{
		Type:       activity.TypeWorldDestroyed,
		Actor:      "architect",
		Verb:       "destroyed",
		Target:     worldID,
		Phrase:     activity.PhraseWorldDestroyed(worldID, uptime),
		WorldID:    worldID,
		DurationMs: duration.Milliseconds(),
	})

	return u, nil
}

// DestroyAll stops and removes all worlds sequentially.
// Returns the list of destroyed worlds and the first error encountered (if any).
func (a *Architect) DestroyAll(ctx context.Context) ([]*models.World, error) {
	worlds, err := a.state.List()
	if err != nil {
		return nil, err
	}

	var destroyed []*models.World
	for _, w := range worlds {
		u, err := a.Destroy(ctx, w.ID)
		if err != nil {
			log.Printf("warning: failed to destroy world %s: %v", w.ID, err)
			continue
		}
		destroyed = append(destroyed, u)
	}
	return destroyed, nil
}
