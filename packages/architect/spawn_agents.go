package architect

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/compile"
	claudecode "spwn.sh/packages/compile/runtimes/claude_code"
	"spwn.sh/packages/image/backend"
	"spwn.sh/packages/world/models"
	"spwn.sh/packages/world/runtime"
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
// that land in durable memory layers (journal/playbooks/skills)
// get synced back on graceful shutdown and re-seeded fresh next
// spawn. Knowledge is world-scoped (bind-mounted from
// spwn/worlds/<name>/knowledge/), so it persists directly and does
// not ride through the agent sync pipeline.
//
// The runtime lookup is hardcoded to claude-code because every
// world spawn today installs spwn:claude-code as a required tool.
// When the runtime becomes a per-world choice this should resolve
// off the world manifest.
func writeRuntimeDefaultConfig(ctx context.Context, be backend.Backend, containerID string, agentHomes map[string]string) error {
	rt, err := runtime.Get("claude-code")
	if err != nil {
		return fmt.Errorf("unknown runtime: %w", err)
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

// rosterColony adapts an agent record list into the claudecode
// ColonyAgentSpec list (used by colony.go's GenerateRoster call
// for roster regeneration on hot-deploy). New code should build
// a compile.Input instead.
func rosterColony(recs []models.AgentRecord) []claudecode.ColonyAgentSpec {
	out := make([]claudecode.ColonyAgentSpec, 0, len(recs))
	for _, r := range recs {
		out = append(out, claudecode.ColonyAgentSpec{Name: r.Name, Role: r.Role})
	}
	return out
}

// rosterCompileAgents projects the world record's agent list onto
// the compile.Input shape.
func rosterCompileAgents(recs []models.AgentRecord) []compile.AgentInput {
	out := make([]compile.AgentInput, 0, len(recs))
	for _, r := range recs {
		out = append(out, compile.AgentInput{Name: r.Name, Role: r.Role})
	}
	return out
}
