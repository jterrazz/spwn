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

// agentHomesForSpawn returns the agentName → containerHomePath map
// for every agent attached to this spawn. Single-agent worlds
// return one entry; multi-agent worlds return one per agent. The
// home paths are the in-container view (/agents/<name>). Each has
// a mirror under <project>/spwn/agents/<name>/ on the host, but the
// two are NOT a live bind mount — spawn copies host→container and
// graceful shutdown copies an allowlisted subset back.
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

// initAgentDeploymentDirs creates the empty per-agent per-world
// filesystem skeleton inside the agent's persistent home:
//
//	~/.spwn/agents/<name>/worlds/<world-id>/
//	  inbox/   - messages received in this world
//	  outbox/  - messages I sent (audit trail)
//	  notes/   - private notes for this world's project
//
// The content-bearing files (role.md, CLAUDE.md) are produced by
// the compiler and materialised alongside these dirs via
// deploy.MaterialiseTree. Hot-deploy uses the same helper.
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

// writeRuntimeDefaultConfig writes the runtime provider's default
// config files into each agent's HOME inside the running container
// via docker cp. Under the docker-cp-not-bind-mount architecture
// the /agents tree is copied into the container at start, so
// dotfiles must also be delivered via docker cp — there's no
// shared host path to write to.
//
// Each spawn re-seeds the default config (safe because the
// container is ephemeral). Any user customisations to .claude.json
// that land in durable memory layers (journal/playbooks) get synced
// back on graceful shutdown and re-seeded fresh next spawn.
// Knowledge is world-scoped (bind-mounted from
// spwn/worlds/<name>/knowledge/), so it persists directly and does
// not ride through the agent sync pipeline.
//
// runtimeName selects the spawn adapter whose DefaultConfigFiles are
// materialised for each agent. Adapters without a Spawn are skipped
// silently (runtimes that don't need pre-seeded container config).
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

// rosterCompileAgents projects the world record's agent list onto
// the transpile.Input shape. For each agent, walk the host-side
// home (agent.AgentDir(name)) and populate everything a renderer
// might need: promoted-playbook index, SOUL.md body, user-authored
// AGENTS.md body. Missing files and malformed frontmatter are
// tolerated silently — `spwn check` is the authoring-side gate;
// spawn is best-effort.
//
// The bodies are there for renderers that can't @-import (codex),
// which inline identity + task into a single AGENTS.md at the
// agent's cwd. Renderers that @-import (claude-code) ignore the
// bodies; the files survive independently on the docker-cp'd agent
// home, so the @-references still resolve.
func rosterCompileAgents(recs []models.AgentRecord) []transpile.AgentInput {
	out := make([]transpile.AgentInput, 0, len(recs))
	for _, r := range recs {
		home := agent.AgentDir(r.Name)
		out = append(out, transpile.AgentInput{
			Name:      r.Name,
			Role:      r.Role,
			Soul:      readAgentFile(home, "SOUL.md"),
			AgentMD:   readAgentFile(home, "AGENTS.md"),
			Playbooks: loadAgentPlaybookIndex(r.Name),
		})
	}
	return out
}

// readAgentFile returns the body of a file relative to the agent's
// home. Missing files return nil (not an error) so renderers can
// tolerate partial scaffolds; any other I/O error surfaces as an
// empty result rather than crashing the spawn — the renderer still
// has enough to produce a usable prompt.
func readAgentFile(home, name string) []byte {
	b, err := os.ReadFile(filepath.Join(home, name))
	if err != nil {
		return nil
	}
	return b
}

// loadAgentPlaybookIndex reads the agent's host-side playbooks dir
// and returns the subset of files that carry valid `name:` +
// `description:` frontmatter. Returns nil when the dir is absent or
// empty. Kept inline here (not in packages/transpile/source) because
// the spawn path doesn't go through ProjectSource — it resolves
// agent dirs directly via packages/agent.AgentDir.
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
