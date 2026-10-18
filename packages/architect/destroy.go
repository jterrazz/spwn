package architect

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"spwn.sh/packages/activity"
	"spwn.sh/packages/agent"
	"spwn.sh/packages/architect/internal/deploy"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/world/models"
)

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
// memory layers (journal, playbooks) are snapshotted out via docker
// cp into the host-side spwn/agents/<name>/ tree. Runtime state that
// isn't in the allowlist (dotfiles, npm caches, compiled CLAUDE.md)
// stays in the container and is discarded with it. Non-graceful
// shutdowns (crash, docker kill) skip this step and lose any unsaved
// memory writes.
func (a *Architect) Destroy(ctx context.Context, worldID string) (*models.World, error) {
	u, err := a.rstate.Get(worldID)
	if err != nil {
		return nil, err
	}

	agentHomes := map[string]string{}
	for _, rec := range u.Agents {
		if rec.Name != "" {
			agentHomes[rec.Name] = "/agents/" + rec.Name
		}
	}
	if len(agentHomes) == 0 && u.Agent != "" {
		agentHomes[u.Agent] = "/agents/" + u.Agent
	}

	// invariant: Sync memory layers out of the container before stopping it.
	// Best-effort: warnings surface via log but never block destroy.
	if len(agentHomes) > 0 {
		for _, w := range deploy.SyncOut(ctx, a.backend, u.ContainerID, agentHomes) {
			log.Printf("warning: %s", w)
		}
	}

	if err := a.backend.Stop(ctx, u.ContainerID); err != nil {
		log.Printf("warning: stop container %s: %v", u.ContainerID, err)
	}
	if err := a.backend.Remove(ctx, u.ContainerID); err != nil {
		log.Printf("warning: remove container %s: %v", u.ContainerID, err)
	}

	duration := time.Since(u.CreatedAt)
	for name := range agentHomes {
		agentPath := filepath.Join(platform.AgentsDir(), name)
		if journalErr := agent.AppendJournal(agentPath, worldID, -1, duration); journalErr != nil {
			log.Printf("warning: failed to write journal for agent %s on destroy: %v", name, journalErr)
		}
	}

	if err := a.rstate.Delete(worldID); err != nil {
		log.Printf("warning: delete rstate for %s: %v", worldID, err)
	}

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
	worlds, err := a.rstate.List()
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
