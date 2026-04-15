package claude_code

import (
	"embed"
	"io/fs"

	ib "spwn.sh/packages/image"
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
		// Note: first-run UI dismissal (hasCompletedOnboarding,
		// trust dialogs, skipDangerousModePermissionPrompt) used to
		// live here as UserCommands but the generated files landed
		// at /home/spwn/.claude.json - the wrong HOME once every
		// agent mounts its own /agents/<name>/ at spawn time. That
		// logic moved to
		// packages/world/internal/runtime/claude.DefaultConfigFiles,
		// which writes the files directly into the per-agent home
		// via the /agents bind mount at spawn time so they land
		// under the HOME the runtime actually reads.
	}
}

func (*tool) Verify() []string {
	return []string{"command -v claude"}
}

func (*tool) Skills() fs.FS {
	sub, _ := fs.Sub(skills, "skills")
	return sub
}
