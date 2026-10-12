package architect

import (
	"fmt"

	"spwn.sh/packages/runtimes"
	"spwn.sh/packages/world/models"
)

const defaultRuntimeName = "claude-code"

func runtimeBackendTool(runtimeName string) string {
	switch runtimeName {
	case "codex":
		return "spwn:codex"
	case "claude-code", "":
		return "spwn:claude-code"
	default:
		return "spwn:claude-code"
	}
}

func runtimePromptFile(runtimeName string) string {
	switch runtimeName {
	case "codex":
		return "AGENTS.md"
	case "claude-code", "":
		return "CLAUDE.md"
	default:
		return "prompt"
	}
}

func runtimeProvider(runtimeName string) string {
	if runtimeName == "" {
		runtimeName = defaultRuntimeName
	}
	adapter, ok := runtimes.Get(runtimeName)
	if !ok {
		return ""
	}
	return adapter.DefaultProvider
}

func resolveRuntimeName(u *models.World) string {
	if u != nil && u.Runtime != "" {
		return u.Runtime
	}
	return defaultRuntimeName
}

func (a *Architect) resolveSpawner(u *models.World) (runtimes.Spawner, error) {
	name := resolveRuntimeName(u)
	rt, err := runtimes.GetSpawner(name)
	if err != nil || rt == nil {
		return nil, fmt.Errorf("world runtime %q is not registered: %w", name, err)
	}
	return rt, nil
}
