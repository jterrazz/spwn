package architect

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/runtimes"
	"spwn.sh/packages/transpile"
	"spwn.sh/packages/world/models"
)

func agentHomesForSpawn(opts SpawnOpts) map[string]string {
	homes := map[string]string{}
	if opts.AgentName != "" {
		homes[opts.AgentName] = "/agents/" + opts.AgentName
	}
	for _, a := range opts.Agents {
		if a.Name == "" {
			continue
		}
		homes[a.Name] = "/agents/" + a.Name
	}
	return homes
}

func initAgentDeploymentDirs(rec models.AgentRecord, worldID string) error {
	agentDir := agent.AgentDir(rec.Name)
	deploymentDir := filepath.Join(agentDir, "worlds", worldID)
	for _, sub := range []string{"inbox", "outbox", "notes"} {
		if err := os.MkdirAll(filepath.Join(deploymentDir, sub), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", sub, err)
		}
	}
	return nil
}

func writeRuntimeDefaultConfig(ctx context.Context, be backend.Backend, containerID, runtimeName string, agentHomes map[string]string) error {
	rt, err := runtimes.GetSpawner(runtimeName)
	if err != nil {
		return fmt.Errorf("unknown runtime %q: %w", runtimeName, err)
	}

	for _, agentHome := range agentHomes {
		files := rt.DefaultConfigFiles(agentHome)
		if len(files) == 0 {
			continue
		}
		for relPath, content := range files {
			absPath := agentHome + "/" + relPath
			if err := be.CopyTo(ctx, containerID, absPath, content); err != nil {
				return fmt.Errorf("cp %s to container: %w", absPath, err)
			}
		}
	}
	return nil
}

func rosterCompileAgents(recs []models.AgentRecord, hookPool []transpile.HookEntry, commandPool []transpile.CommandEntry) []transpile.AgentInput {
	out := make([]transpile.AgentInput, 0, len(recs))
	for _, r := range recs {
		home := agent.AgentDir(r.Name)
		model, provider := loadAgentRuntimePrefs(r.Name)
		out = append(out, transpile.AgentInput{
			Name:      r.Name,
			Role:      r.Role,
			Soul:      readAgentFile(home, "SOUL.md"),
			AgentMD:   readAgentFile(home, "AGENTS.md"),
			Playbooks: loadAgentPlaybookIndex(r.Name),
			Model:     model,
			Provider:  provider,
			Hooks:     agentHookSelection(r.Name, hookPool),
			Commands:  agentCommandSelection(r.Name, commandPool),
		})
	}
	return out
}

func loadAgentRuntimePrefs(name string) (model, provider string) {
	m, err := agent.LoadManifest(name)
	if err != nil || m == nil {
		return "", ""
	}
	return m.Runtime.Model, m.Runtime.Provider
}

func readAgentFile(home, name string) []byte {
	b, err := os.ReadFile(filepath.Join(home, name))
	if err != nil {
		return nil
	}
	return b
}

func loadAgentPlaybookIndex(agentName string) []transpile.PlaybookEntry {
	playbooksDir := filepath.Join(agent.AgentDir(agentName), "playbooks")
	entries, err := os.ReadDir(playbooksDir)
	if err != nil {
		return nil
	}
	var out []transpile.PlaybookEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		body, err := os.ReadFile(filepath.Join(playbooksDir, e.Name()))
		if err != nil {
			continue
		}
		entry, ok := parsePlaybookHeader(body)
		if !ok {
			continue
		}
		out = append(out, entry)
	}
	return out
}
