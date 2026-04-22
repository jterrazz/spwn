package claudecode

import (
	"spwn.sh/packages/dependency/tool"
	"io/fs"
)


// Tool is the spwn:claude-code tool - Claude Code AI agent runtime.
var Tool = &claudeCodeTool{}

type claudeCodeTool struct{}

func (*claudeCodeTool) Name() string    { return "spwn:claude-code" }
func (*claudeCodeTool) Version() string { return "latest" }

// Dependencies: only spwn:unix for curl + jq. We used to also
// require spwn:node because the old install path was
// `npm install -g @anthropic-ai/claude-code`; the native installer
// below ships a self-contained binary so node is no longer part
// of the required world footprint.
func (*claudeCodeTool) Dependencies() []string { return []string{"spwn:unix"} }

func (*claudeCodeTool) Install() tool.InstallSpec {
	return tool.InstallSpec{
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
		// First-run UI dismissal (hasCompletedOnboarding, trust
		// dialogs, skipDangerousModePermissionPrompt) belongs at
		// spawn time, not image-build time — the image's /home/spwn
		// is not the agent's actual HOME (/agents/<name>/). See
		// spawn.go's DefaultConfigFiles for the live path.
	}
}

func (*claudeCodeTool) Verify() []string {
	return []string{"command -v claude"}
}

func (*claudeCodeTool) Skills() fs.FS { return nil }
