package codex

import (
	"spwn.sh/packages/dependency/tool"
	"io/fs"
)


// Tool is the spwn:codex tool - OpenAI Codex agent runtime.
var Tool = &codexTool{}

type codexTool struct{}

func (*codexTool) Name() string           { return "spwn:codex" }
func (*codexTool) Version() string        { return "latest" }
func (*codexTool) Dependencies() []string { return []string{"spwn:node"} }

func (*codexTool) Install() tool.InstallSpec {
	// Per-agent config (model pin, trust, warning suppression) lives
	// in the agent's HOME (/agents/<name>/.codex/config.toml), written
	// at spawn time by GenerateAgentConfigTOML + PrelaunchShell — not
	// here at image-build time where $HOME is /home/spwn and the
	// running agent never reads it.
	return tool.InstallSpec{
		Commands: []string{
			"npm install -g @openai/codex",
		},
	}
}

func (*codexTool) Verify() []string {
	return []string{"command -v codex"}
}

func (*codexTool) Skills() fs.FS { return nil }
