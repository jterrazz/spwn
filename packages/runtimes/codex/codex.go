package codex

import (
	"io/fs"
	"spwn.sh/packages/dependency/tool"
)

// Tool is the spwn:codex tool - OpenAI Codex agent runtime.
var Tool = &codexTool{}

type codexTool struct{}

func (*codexTool) Name() string           { return "spwn:codex" }
func (*codexTool) Version() string        { return "latest" }
func (*codexTool) Dependencies() []string { return []string{"spwn:node"} }

const codexVersion = "0.122.0"

func (*codexTool) Install() tool.InstallSpec {
	return tool.InstallSpec{
		Commands: []string{
			"npm install -g @openai/codex@" + codexVersion,
		},
	}
}

func (*codexTool) Verify() []string {
	return []string{"command -v codex"}
}

func (*codexTool) Skills() fs.FS { return nil }
