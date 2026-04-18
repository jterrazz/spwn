package codex

import (
	"spwn.sh/packages/dependency"
	"io/fs"
)


// Tool is the spwn:codex tool - OpenAI Codex agent runtime.
var Tool = &tool{}

type tool struct{}

func (*tool) Name() string           { return "spwn:codex" }
func (*tool) Kind() dependency.Kind          { return dependency.KindRuntime }
func (*tool) Version() string        { return "latest" }
func (*tool) Dependencies() []string { return []string{"spwn:node"} }

func (*tool) Install() dependency.InstallSpec {
	return dependency.InstallSpec{
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

func (*tool) Verify() []string {
	return []string{"command -v codex"}
}

func (*tool) Skills() fs.FS { return nil }

func (*tool) Runtimes() []string         { return nil }
func (*tool) Config(runtime string) []byte { return nil }
