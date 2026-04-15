package codex

import (
	"embed"
	"io/fs"

	ib "spwn.sh/packages/image"
)

//go:embed skills/*
var skills embed.FS

// Tool is the @spwn/codex tool - OpenAI Codex agent runtime.
var Tool = &tool{}

type tool struct{}

func (*tool) Name() string           { return "@spwn/codex" }
func (*tool) Kind() ib.Kind          { return ib.KindRuntime }
func (*tool) Version() string        { return "latest" }
func (*tool) Dependencies() []string { return []string{"@spwn/node"} }

func (*tool) Install() ib.InstallSpec {
	return ib.InstallSpec{
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

func (*tool) Skills() fs.FS {
	sub, _ := fs.Sub(skills, "skills")
	return sub
}

func (*tool) Runtimes() []string         { return nil }
func (*tool) Config(runtime string) []byte { return nil }
