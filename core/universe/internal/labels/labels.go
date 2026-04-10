// Package labels defines the canonical Docker container labels that
// spwn uses to identify and reconstruct worlds.
//
// Container labels are spwn's source of truth for world existence and
// immutable creation-time metadata. The mutable bits (deployed agent
// list, runtime session ids) live in per-world JSON files under
// ~/.spwn/runtime/<world-id>.json — see the runtimestate package.
//
// The split is deliberate: labels are immutable once a container is
// created, so anything that needs to change after spawn must live
// elsewhere. The advantage is that listing worlds becomes a single
// `docker ps --filter label=sh.spwn.kind=world` and there is zero
// possibility of state-file drift when the user runs `docker rm`.
package labels

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"spwn.sh/core/universe/internal/models"
)

// All spwn labels share this prefix so they're easy to spot in
// `docker inspect` output and easy to filter on.
const Prefix = "sh.spwn."

// Kind enumerates the categories of containers spwn creates.
const (
	KindKey       = Prefix + "kind"
	KindWorld     = "world"
	KindArchitect = "architect"
)

// World metadata keys. These map 1:1 onto the immutable fields of
// models.World. Anything that can change after creation lives in
// runtimestate, not here.
const (
	WorldID           = Prefix + "world.id"
	WorldName         = Prefix + "world.name"
	WorldConfig       = Prefix + "world.config"
	WorldAgent        = Prefix + "world.agent"        // primary/legacy single-agent name
	WorldAgentID      = Prefix + "world.agent_id"     // primary/legacy single-agent id
	WorldRuntime      = Prefix + "world.runtime"
	WorldOrganization = Prefix + "world.organization"
	WorldMindPath     = Prefix + "world.mind_path"
	WorldGateDir      = Prefix + "world.gate_dir"
	WorldWorkspaces   = Prefix + "world.workspaces" // JSON-encoded []models.Workspace
	WorldAgents       = Prefix + "world.agents"     // JSON-encoded []models.AgentRecord (creation-time only)
	WorldCreatedAt    = Prefix + "world.created_at" // RFC3339
)

// WorldLabels builds the full label map for a world container. The
// caller passes this to ContainerConfig.Labels at create time.
//
// Empty fields are omitted so we don't pollute container metadata with
// sentinel "unset" strings. JSON-encoded fields (workspaces, agents)
// are only included when non-empty.
func WorldLabels(w models.World) map[string]string {
	out := map[string]string{
		KindKey:        KindWorld,
		WorldID:        w.ID,
		WorldConfig:    w.Config,
		WorldCreatedAt: w.CreatedAt.UTC().Format(time.RFC3339),
	}
	if w.Name != "" {
		out[WorldName] = w.Name
	}
	if w.Agent != "" {
		out[WorldAgent] = w.Agent
	}
	if w.AgentID != "" {
		out[WorldAgentID] = w.AgentID
	}
	if w.Runtime != "" {
		out[WorldRuntime] = w.Runtime
	}
	if w.Organization != "" {
		out[WorldOrganization] = w.Organization
	}
	if w.MindPath != "" {
		out[WorldMindPath] = w.MindPath
	}
	if w.GateDir != "" {
		out[WorldGateDir] = w.GateDir
	}
	if len(w.Workspaces) > 0 {
		if data, err := json.Marshal(w.Workspaces); err == nil {
			out[WorldWorkspaces] = string(data)
		}
	}
	if len(w.Agents) > 0 {
		if data, err := json.Marshal(w.Agents); err == nil {
			out[WorldAgents] = string(data)
		}
	}
	return out
}

// ParseWorld reconstructs a models.World from a Docker container's
// labels. Returns an error if the labels do not contain a valid spwn
// world marker. Caller is responsible for filling in fields that come
// from the container itself (ContainerID, Status from container state).
func ParseWorld(lbls map[string]string) (models.World, error) {
	if lbls == nil {
		return models.World{}, fmt.Errorf("nil labels")
	}
	if lbls[KindKey] != KindWorld {
		return models.World{}, fmt.Errorf("not a spwn world container (kind=%q)", lbls[KindKey])
	}
	id := lbls[WorldID]
	if id == "" {
		return models.World{}, fmt.Errorf("missing %s", WorldID)
	}

	w := models.World{
		ID:           id,
		Name:         lbls[WorldName],
		Config:       lbls[WorldConfig],
		Agent:        lbls[WorldAgent],
		AgentID:      lbls[WorldAgentID],
		Runtime:      lbls[WorldRuntime],
		Organization: lbls[WorldOrganization],
		MindPath:     lbls[WorldMindPath],
		GateDir:      lbls[WorldGateDir],
	}

	if raw := lbls[WorldCreatedAt]; raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			w.CreatedAt = t
		}
	}
	if raw := lbls[WorldWorkspaces]; raw != "" {
		var ws []models.Workspace
		if err := json.Unmarshal([]byte(raw), &ws); err == nil {
			w.Workspaces = ws
		}
	}
	if raw := lbls[WorldAgents]; raw != "" {
		var agents []models.AgentRecord
		if err := json.Unmarshal([]byte(raw), &agents); err == nil {
			w.Agents = agents
		}
	}

	return w, nil
}

// IsWorld reports whether a label map identifies a spwn world container.
func IsWorld(lbls map[string]string) bool {
	return lbls != nil && lbls[KindKey] == KindWorld
}

// IsArchitect reports whether a label map identifies the spwn architect
// daemon container.
func IsArchitect(lbls map[string]string) bool {
	return lbls != nil && lbls[KindKey] == KindArchitect
}

// SortKeysForDebug returns the spwn-prefixed labels of a map in
// alphabetical order. Used by debug output and tests where we want a
// stable representation regardless of map iteration order.
func SortKeysForDebug(lbls map[string]string) []string {
	keys := []string{}
	for k := range lbls {
		if strings.HasPrefix(k, Prefix) {
			keys = append(keys, k)
		}
	}
	// stdlib sort would pull in another import; keep it local since
	// the list is tiny (≤ 12 entries).
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}
