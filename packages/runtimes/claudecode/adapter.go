package claudecode

import "spwn.sh/packages/runtimes"

// Adapter is the claude-code umbrella: install recipe + render + spawn.
// All three facets are implemented, so the Adapter fills every field.
var Adapter = runtimes.Adapter{
	Name:            "claude-code",
	CatalogRef:      "spwn:claude-code",
	DefaultProvider: "anthropic",
	Tool:            Tool,
	Render:          Renderer,
	Spawn:           Spawner,
}

func init() { runtimes.Register(Adapter) }
