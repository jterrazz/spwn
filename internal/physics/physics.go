package physics

import (
	"fmt"
	"strings"

	"github.com/jterrazz/universe/internal/config"
)

// Generate produces the physics.md content for a universe.
func Generate(cfg *config.UniverseConfig) string {
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
  - bash
  - git
  - node
  - python3
  - curl
  - claude (Claude Code CLI)

## Laws

1. The filesystem is ephemeral — only /workspace and /mind persist across restarts.
2. Network access is available but may be restricted in future phases.
3. The agent operates within the resource limits defined in Constants.
4. /mind is shared identity — treat it as long-term memory across universes.
5. /workspace is the project context — the current task lives here.
`, memory, cpu, timeoutStr, cfg.Image, mountSection)
}
