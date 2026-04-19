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
	return tool.InstallSpec{
		Commands: []string{
			"npm install -g @openai/codex",
		},
		// User-level config: runs after USER switch.
		// Pre-configure codex to trust /workspace and skip sandbox prompts.
		UserCommands: []string{
			`mkdir -p {{.Home}}/.codex && printf 'model = "gpt-5.4"\n\n[projects."/"]\ntrust_level = "trusted"\n\n[notice]\nhide_full_access_warning = true\n' > {{.Home}}/.codex/config.toml`,
		},
	}
}

func (*codexTool) Verify() []string {
	return []string{"command -v codex"}
}

func (*codexTool) Skills() fs.FS { return nil }
