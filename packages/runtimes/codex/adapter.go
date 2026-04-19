package codex

import "spwn.sh/packages/runtimes"

// Adapter is the codex umbrella. Install recipe + spawn-time
// prelaunch (OpenAI auth symlink); no renderer — codex has no
// source→Tree compiler yet.
var Adapter = runtimes.Adapter{
	Name:            "codex",
	DefaultProvider: "openai",
	Tool:            Tool,
	Spawn:           Spawner,
}

func init() { runtimes.Register(Adapter) }
