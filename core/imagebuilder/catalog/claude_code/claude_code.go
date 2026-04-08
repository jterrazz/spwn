package claude_code

import (
	"embed"
	"io/fs"

	ib "spwn.sh/core/imagebuilder"
)

//go:embed skills/*
var skills embed.FS

// Tool is the @claude-code tool — Claude Code AI agent runtime.
var Tool = &tool{}

type tool struct{}

func (*tool) Name() string           { return "@claude-code" }
func (*tool) Kind() ib.Kind          { return ib.KindRuntime }
func (*tool) Version() string        { return "latest" }
func (*tool) Dependencies() []string { return []string{"@node"} }

func (*tool) Install() ib.InstallSpec {
	return ib.InstallSpec{
		Commands: []string{
			"npm install -g @anthropic-ai/claude-code",
			// Pre-configure Claude Code (onboarding + workspace trust + permission skip)
			"mkdir -p /home/spwn/.claude",
			`echo '{"hasCompletedOnboarding":true,"projects":{"/workspace":{"hasTrustDialogAccepted":true},"/home/spwn":{"hasTrustDialogAccepted":true}}}' > /home/spwn/.claude.json`,
			`echo '{"skipDangerousModePermissionPrompt":true}' > /home/spwn/.claude/settings.json`,
			"chown -R spwn:spwn /home/spwn/.claude.json /home/spwn/.claude",
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
