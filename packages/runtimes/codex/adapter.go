package codex

import "spwn.sh/packages/runtimes"

// Adapter is the codex umbrella: install recipe + spawn-time
// (credential sync, prelaunch shell, one-shot flag synthesis,
// output parsing) + source→Tree renderer. Fully first-class — same
// surface as claude-code.
var Adapter = runtimes.Adapter{
	Name:            "codex",
	DefaultProvider: "openai",
	Tool:            Tool,
	Render:          Renderer,
	Spawn:           Spawner,
}

func init() { runtimes.Register(Adapter) }
