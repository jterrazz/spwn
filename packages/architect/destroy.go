package architect

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/activity"
	"spwn.sh/packages/world/models"
	"spwn.sh/packages/platform"
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
//
// Before the container is stopped, every deployed agent's durable
// memory layers (journal, knowledge, playbooks, skills) are
// snapshotted out via docker cp into the host-side spwn/agents/<name>/
// tree. Runtime state that isn't in the allowlist — dotfiles, npm
// caches, compiled CLAUDE.md — stays in the container and is
// discarded with it. Non-graceful shutdowns (crash, docker kill)
// skip this step and lose any unsaved memory writes.
func (a *Architect) Destroy(ctx context.Context, worldID string) (*models.World, error) {
	u, err := a.state.Get(worldID)
	if err != nil {
		return nil, err
	}

	// Collect the agent home mapping so we can sync out before the
	// container is stopped. Both multi-agent (Agents list) and
	// legacy single-agent (Agent scalar) shapes feed the same map.
	agentHomes := map[string]string{}
	for _, rec := range u.Agents {
		if rec.Name != "" {
			agentHomes[rec.Name] = "/agents/" + rec.Name
		}
	}
	if len(agentHomes) == 0 && u.Agent != "" {
		agentHomes[u.Agent] = "/agents/" + u.Agent
	}

	// Sync memory layers out of the container before stopping it.
	// Best-effort: warnings surface via log but never block destroy.
	if len(agentHomes) > 0 {
		for _, w := range syncAgentsOutOf(ctx, a.backend, u.ContainerID, agentHomes) {
			log.Printf("warning: %s", w)
		}
	}

	a.backend.Stop(ctx, u.ContainerID)
	a.backend.Remove(ctx, u.ContainerID)

	// Write a journal entry for every agent that was deployed in
	// this world. Happens after the sync-out so the new entry joins
	// the already-freshened on-disk journal, not a stale copy.
	duration := time.Since(u.CreatedAt)
	for name := range agentHomes {
		agentPath := filepath.Join(platform.AgentsDir(), name)
		if journalErr := agent.AppendJournal(agentPath, worldID, -1, duration); journalErr != nil {
			log.Printf("warning: failed to write journal for agent %s on destroy: %v", name, journalErr)
		}
	}

	a.state.Delete(worldID)

	// Emit activity events
	uptime := formatUptime(duration)
	for name := range agentHomes {
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
