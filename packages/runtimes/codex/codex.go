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

// codexVersion is the pinned @openai/codex version the image
// Installs. Pinning keeps every user's container on the same codex
// Binary regardless of when their Docker image was last built — the
// Runtime adapter passes --skip-git-repo-check +
// --dangerously-bypass-approvals-and-sandbox, which only exist from
// 0.122 onwards. An unpinned `latest` led to users on older machines
// Getting a cached pre-0.122 codex that rejects those flags. Bump
// This string deliberately when we need features from a newer codex.
const codexVersion = "0.122.0"

func (*codexTool) Install() tool.InstallSpec {
	// Per-agent config (model pin, trust, warning suppression) lives
	// In the agent's HOME (/agents/<name>/.codex/config.toml), written
	// At spawn time by GenerateAgentConfigTOML + PrelaunchShell — not
	// Here at image-build time where $HOME is /home/spwn and the
	// Running agent never reads it.
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
