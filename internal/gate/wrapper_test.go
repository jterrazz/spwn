package gate

import (
	"strings"
	"testing"

	"github.com/jterrazz/universe/internal/config"
)

func TestWrapperScript_ContainsShebang(t *testing.T) {
	ia := config.Interaction{Source: "mcp/test", As: "test-cmd"}
	script := WrapperScript(ia)

	if !strings.HasPrefix(script, "#!/bin/sh") {
		t.Error("wrapper script should start with shebang")
	}
}

func TestWrapperScript_ContainsInteractionName(t *testing.T) {
	ia := config.Interaction{Source: "mcp/slack", As: "slack-send"}
	script := WrapperScript(ia)

	if !strings.Contains(script, "slack-send") {
		t.Error("wrapper script should contain interaction name")
	}
}

func TestWrapperScript_ContainsSocketPath(t *testing.T) {
	ia := config.Interaction{Source: "mcp/db", As: "db-query"}
	script := WrapperScript(ia)

	if !strings.Contains(script, "gate.sock") {
		t.Error("wrapper script should reference gate.sock")
	}
}

func TestAllWrapperCommands_CreatesDirectory(t *testing.T) {
	interactions := []config.Interaction{
		{Source: "mcp/a", As: "cmd-a"},
	}
	cmds := AllWrapperCommands(interactions)

	if len(cmds) == 0 {
		t.Fatal("expected commands")
	}
	if cmds[0] != "mkdir -p /gate/bin" {
		t.Errorf("first command should create /gate/bin, got: %s", cmds[0])
	}
}

func TestAllWrapperCommands_IncludesAllInteractions(t *testing.T) {
	interactions := []config.Interaction{
		{Source: "mcp/a", As: "cmd-a"},
		{Source: "mcp/b", As: "cmd-b"},
	}
	cmds := AllWrapperCommands(interactions)

	joined := strings.Join(cmds, "\n")
	if !strings.Contains(joined, "cmd-a") {
		t.Error("commands should reference cmd-a")
	}
	if !strings.Contains(joined, "cmd-b") {
		t.Error("commands should reference cmd-b")
	}
}

func TestAllWrapperCommands_AddsToPath(t *testing.T) {
	interactions := []config.Interaction{
		{Source: "mcp/a", As: "cmd-a"},
	}
	cmds := AllWrapperCommands(interactions)

	joined := strings.Join(cmds, "\n")
	if !strings.Contains(joined, "/etc/profile.d/gate.sh") {
		t.Error("commands should set up PATH via profile.d")
	}
}

func TestAllWrapperCommands_SymlinksToUsrLocalBin(t *testing.T) {
	interactions := []config.Interaction{
		{Source: "mcp/a", As: "cmd-a"},
	}
	cmds := AllWrapperCommands(interactions)

	joined := strings.Join(cmds, "\n")
	if !strings.Contains(joined, "/usr/local/bin/cmd-a") {
		t.Error("commands should symlink to /usr/local/bin")
	}
}
