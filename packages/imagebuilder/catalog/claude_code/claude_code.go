package claude_code

import (
	"embed"
	"io/fs"

	ib "spwn.sh/packages/imagebuilder"
)

//go:embed skills/*
var skills embed.FS

// Tool is the @spwn/claude-code tool - Claude Code AI agent runtime.
var Tool = &tool{}

type tool struct{}

func (*tool) Name() string           { return "@spwn/claude-code" }
func (*tool) Kind() ib.Kind          { return ib.KindRuntime }
func (*tool) Version() string        { return "latest" }
func (*tool) Dependencies() []string { return []string{"@spwn/node"} }

func (*tool) Install() ib.InstallSpec {
	return ib.InstallSpec{
		Commands: []string{
			"npm install -g @anthropic-ai/claude-code",
		},
		// User-level config: runs after USER switch.
		// {{.Home}} and {{.User}} are templated by the generator.
		UserCommands: []string{
			`mkdir -p {{.Home}}/.claude && echo '{"hasCompletedOnboarding":true,"projects":{"/workspace":{"hasTrustDialogAccepted":true},"{{.Home}}":{"hasTrustDialogAccepted":true}}}' > {{.Home}}/.claude.json && echo '{"skipDangerousModePermissionPrompt":true}' > {{.Home}}/.claude/settings.json`,
		},
	}
}

func (*tool) Verify() []string {
	return []string{"command -v claude"}
}

func (*tool) Skills() fs.FS {
	sub, _ := fs.Sub(skills, "skills")
	return sub
}
