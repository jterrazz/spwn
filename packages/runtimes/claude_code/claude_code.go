package claude_code

import (
	"spwn.sh/packages/dependency"
	"io/fs"

	ib "spwn.sh/packages/image"
)


// Tool is the spwn:claude-code tool - Claude Code AI agent runtime.
var Tool = &tool{}

type tool struct{}

func (*tool) Name() string    { return "spwn:claude-code" }
func (*tool) Kind() dependency.Kind   { return dependency.KindRuntime }
func (*tool) Version() string { return "latest" }

// Dependencies: only spwn:unix for curl + jq. We used to also
// require spwn:node because the old install path was
// `npm install -g @anthropic-ai/claude-code`; the native installer
// below ships a self-contained binary so node is no longer part
// of the required world footprint.
func (*tool) Dependencies() []string { return []string{"spwn:unix"} }

func (*tool) Install() ib.InstallSpec {
	return ib.InstallSpec{
		Commands: []string{
			// Native install: downloads a self-contained binary,
			// no Node.js / npm in the image. The installer drops
			// the binary at $HOME/.local/share/claude/versions/<ver>
			// and places a symlink at $HOME/.local/bin/claude
			// pointing at the current version. $HOME during a
			// Dockerfile RUN step is /root, so those paths resolve
			// under /root/.local.
			//
			// We `cp -L` via the symlink so the destination is the
			// 200+ MB binary itself (not a dangling symlink), then
			// wipe /root/.local so the layer doesn't ship the
			// private staging tree.
			"curl -fsSL https://claude.ai/install.sh | bash",
			"cp -L /root/.local/bin/claude /usr/local/bin/claude",
			"chmod +x /usr/local/bin/claude",
			"rm -rf /root/.local /root/.claude",
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

func (*tool) Skills() fs.FS { return nil }

func (*tool) Runtimes() []string         { return nil }
func (*tool) Config(runtime string) []byte { return nil }
