package claudecode

import (
	"io/fs"
	"spwn.sh/packages/dependency/tool"
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
			"curl -fsSL https://claude.ai/install.sh | bash",
			"cp -L /root/.local/bin/claude /usr/local/bin/claude",
			"chmod +x /usr/local/bin/claude",
			"rm -rf /root/.local /root/.claude",
		},
	}
}

func (*claudeCodeTool) Verify() []string {
	return []string{"command -v claude"}
}

func (*claudeCodeTool) Skills() fs.FS { return nil }
