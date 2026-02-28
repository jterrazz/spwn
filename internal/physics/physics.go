package physics

import (
	"context"
	"fmt"
	"strings"

	"github.com/jterrazz/universe/internal/backend"
	"github.com/jterrazz/universe/internal/config"
)

// DefaultElements is the hardcoded list used when introspection is unavailable.
var DefaultElements = []string{"bash", "git", "node", "python3", "curl", "claude (Claude Code CLI)"}

// ProbeTargets lists binaries to check for in a container.
var ProbeTargets = []string{
	"bash", "sh", "git", "node", "npm", "python3",
	"curl", "wget", "jq", "claude", "go", "rustc", "gcc", "make",
}

// DetectElements probes a running container for common binaries.
func DetectElements(ctx context.Context, b backend.Backend, universeID string) ([]string, error) {
	var found []string
	for _, bin := range ProbeTargets {
		result, err := b.Exec(ctx, universeID, []string{"sh", "-c", "command -v " + bin})
		if err != nil {
			continue
		}
		if result.ExitCode == 0 {
			found = append(found, bin)
		}
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("no elements detected")
	}
	return found, nil
}

// GenerateWithElements produces the physics.md content with a dynamic element list.
func GenerateWithElements(cfg *config.UniverseConfig, elements []string) string {
	memory := cfg.Memory
	if memory == "" {
		memory = "512m"
	}

	cpu := cfg.CPU
	if cpu == 0 {
		cpu = 1.0
	}

	timeoutStr := "none"
	if cfg.Timeout > 0 {
		timeoutStr = cfg.Timeout.String()
	}

	var mounts []string
	if cfg.Mind != "" {
		mounts = append(mounts, "  - /mind — persistent mind storage (personas, skills, knowledge, playbooks, journal)")
	}
	if cfg.Workspace != "" {
		mounts = append(mounts, "  - /workspace — host workspace (read-write)")
	}

	mountSection := "  (none)"
	if len(mounts) > 0 {
		mountSection = strings.Join(mounts, "\n")
	}

	var elementLines []string
	for _, e := range elements {
		elementLines = append(elementLines, "  - "+e)
	}
	elementSection := strings.Join(elementLines, "\n")

	var interactionSection string
	if len(cfg.Interactions) > 0 {
		var lines []string
		for _, ia := range cfg.Interactions {
			desc := ia.Description
			if desc == "" {
				if len(ia.Capabilities) > 0 {
					desc = fmt.Sprintf("Bridge to %s (%s)", ia.Source, strings.Join(ia.Capabilities, ", "))
				} else {
					desc = fmt.Sprintf("Bridge to %s", ia.Source)
				}
			}
			lines = append(lines, fmt.Sprintf("  - `%s` — %s", ia.As, desc))
		}
		interactionSection = fmt.Sprintf("\n## Interactions\n\nAvailable bridges to external services:\n%s\n", strings.Join(lines, "\n"))
	}

	return fmt.Sprintf(`# Physics

This file describes the constants, topology, and laws of this universe.
It is generated automatically and mounted read-only.

## Constants

- memory: %s
- cpu: %.1f
- timeout: %s
- image: %s

## Topology

Mounted volumes:
%s

## Elements

Available tools:
%s
%s
## Laws

1. The filesystem is ephemeral — only /workspace and /mind persist across restarts.
2. Network access is available but may be restricted in future phases.
3. The agent operates within the resource limits defined in Constants.
4. /mind is shared identity — treat it as long-term memory across universes.
5. /workspace is the project context — the current task lives here.
`, memory, cpu, timeoutStr, cfg.Image, mountSection, elementSection, interactionSection)
}

// Generate produces the physics.md content for a universe using default elements.
func Generate(cfg *config.UniverseConfig) string {
	return GenerateWithElements(cfg, DefaultElements)
}
