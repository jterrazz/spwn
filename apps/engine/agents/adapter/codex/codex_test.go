package codex_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/adapter/codex"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const owners = `model = "gpt-5-codex"
approval_policy = "on-request"

[mcp_servers.docs]
command = "npx"
args = ["-y", "docs-mcp"]
`

func settingsFile(t *testing.T, text string) (home, at string) {
	t.Helper()
	home = t.TempDir()
	at = filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(at), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return home, at
}

func hooks(home string) *codex.Hooks {
	return codex.New(local.New(nil, nil, nil, nil).FS, home, contracttest.Morning)
}

func TestCodexKeepsTheHooksContract(t *testing.T) {
	contracttest.Hooks(t, func(t *testing.T) contracttest.HooksSubject {
		home, at := settingsFile(t, owners)
		return contracttest.HooksSubject{
			Hooks: hooks(home),
			OwnersKept: func(t *testing.T) bool {
				data, err := os.ReadFile(at)
				if err != nil {
					t.Fatal(err)
				}
				return strings.HasSuffix(string(data), owners)
			},
			Recorded: []contracttest.Recorded{{
				Name:    "agent-turn-complete",
				Payload: []byte(`{"type":"agent-turn-complete","thread-id":"0199a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b","turn-id":"12","cwd":"/Users/admin/app","input-messages":["Fix the flaky test"],"last-assistant-message":"Done. The flaky test now waits for the server."}`),
				Want:    agents.Event{Kind: agents.TurnDone, Session: "0199a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b", Folder: "/Users/admin/app", Prompt: "Fix the flaky test", Message: "Done. The flaky test now waits for the server."},
			}},
			Unknown: [][]byte{[]byte(`{"type":"approval-requested","thread-id":"t-1"}`), []byte(`{"type":"agent-turn-complete"}`)},
		}
	})
}

func TestANotifyTheOwnerSetIsNeverReplaced(t *testing.T) {
	theirs := "notify = [\n  \"terminal-notifier\",\n  \"-title\", \"Codex\",\n]\n" + owners
	home, at := settingsFile(t, theirs)
	h := hooks(home)
	if err := h.Install(context.Background()); err == nil {
		t.Error("Install replaced the owner's notify")
	}
	if installed, _ := h.Inspect(context.Background()); installed {
		t.Error("the owner's notify read as station's hook")
	}
	if err := h.Remove(context.Background()); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(at); string(data) != theirs {
		t.Errorf("the owner's file changed:\n%s", data)
	}
}

func TestASettingsFileNotThereYetIsMadeWithTheHookAlone(t *testing.T) {
	home := t.TempDir()
	h := hooks(home)
	if err := h.Install(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if string(data) != "notify = [\"station\", \"agents\", \"record\", \"--app\", \"codex\"]\n" {
		t.Errorf("config.toml:\n%s", data)
	}
}
