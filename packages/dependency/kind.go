package dependency

// Kind classifies what role a dependency plays.
type Kind string

const (
	KindRuntime  Kind = "runtime"  // Agent thinking engine (@spwn/claude-code, @spwn/aider)
	KindTool     Kind = "tool"     // Extra capability (@spwn/qmd, @jq)
	KindSDK      Kind = "sdk"      // Language/runtime SDK (@spwn/node, @spwn/python)
	KindPlatform Kind = "platform" // Spwn infrastructure (@spwn/cli, @spwn/architect)
)
