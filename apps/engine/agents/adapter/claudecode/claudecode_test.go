package claudecode_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/adapter/claudecode"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const owners = `{
  "model": "opus",
  "permissions": {"allow": ["Bash(make check)"]},
  "hooks": {
    "Stop": [{"hooks": [{"type": "command", "command": "afplay /System/Library/Sounds/Glass.aiff"}]}]
  }
}
`

func recorded(transcript string) []contracttest.Recorded {
	folder := "/Users/admin/app"
	base := agents.Event{Session: "8f1c2a90-5b7e-4c1d-9a3e-2f6b7c8d9e01", Folder: folder}
	with := func(e agents.Event) agents.Event {
		e.Session, e.Folder = base.Session, base.Folder
		return e
	}
	head := `"session_id":"8f1c2a90-5b7e-4c1d-9a3e-2f6b7c8d9e01","transcript_path":"` + transcript + `","cwd":"/Users/admin/app",`
	return []contracttest.Recorded{
		{Name: "SessionStart", Payload: []byte(`{` + head + `"hook_event_name":"SessionStart","source":"startup"}`), Want: with(agents.Event{Kind: agents.Began})},
		{Name: "UserPromptSubmit", Payload: []byte(`{` + head + `"hook_event_name":"UserPromptSubmit","prompt":"Build the landing page"}`), Want: with(agents.Event{Kind: agents.Prompted, Prompt: "Build the landing page"})},
		{Name: "PreToolUse Edit", Payload: []byte(`{` + head + `"hook_event_name":"PreToolUse","tool_name":"Edit","tool_input":{"file_path":"/Users/admin/app/src/pages/index.astro","old_string":"a\nb","new_string":"a\nb\nc"}}`), Want: with(agents.Event{Kind: agents.UseStarting, Act: agents.ActEdit, Path: folder + "/src/pages/index.astro"})},
		{Name: "PostToolUse Edit", Payload: []byte(`{` + head + `"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/Users/admin/app/src/pages/index.astro","old_string":"a\nb","new_string":"a\nb\nc"},"tool_response":{"filePath":"/Users/admin/app/src/pages/index.astro","success":true}}`), Want: with(agents.Event{Kind: agents.UseDone, Act: agents.ActEdit, Path: folder + "/src/pages/index.astro", Added: 3, Removed: 2})},
		{Name: "PostToolUse MultiEdit", Payload: []byte(`{` + head + `"hook_event_name":"PostToolUse","tool_name":"MultiEdit","tool_input":{"file_path":"/Users/admin/app/a.ts","edits":[{"old_string":"x","new_string":"y\nz"},{"old_string":"p\nq","new_string":""}]}}`), Want: with(agents.Event{Kind: agents.UseDone, Act: agents.ActEdit, Path: folder + "/a.ts", Added: 2, Removed: 3})},
		{Name: "PreToolUse Bash", Payload: []byte(`{` + head + `"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"npx vitest run","description":"Run the tests"}}`), Want: with(agents.Event{Kind: agents.UseStarting, Act: agents.ActRun, Command: "npx vitest run"})},
		{Name: "PostToolUse Bash", Payload: []byte(`{` + head + `"hook_event_name":"PostToolUse","tool_name":"Bash","tool_input":{"command":"npx vitest run"},"tool_response":{"stdout":"Tests  41 passed (58)","stderr":"","interrupted":false}}`), Want: with(agents.Event{Kind: agents.UseDone, Act: agents.ActRun, Command: "npx vitest run", Output: "Tests  41 passed (58)"})},
		{Name: "PreToolUse Task", Payload: []byte(`{` + head + `"hook_event_name":"PreToolUse","tool_name":"Task","tool_input":{"description":"Explore","prompt":"find the auth code"}}`), Want: with(agents.Event{Kind: agents.UseStarting, Act: agents.ActDelegate})},
		{Name: "PreToolUse Grep", Payload: []byte(`{` + head + `"hook_event_name":"PreToolUse","tool_name":"Grep","tool_input":{"pattern":"refresh"}}`), Want: with(agents.Event{Kind: agents.UseStarting, Act: agents.ActOther})},
		{Name: "Notification", Payload: []byte(`{` + head + `"hook_event_name":"Notification","message":"Claude needs your permission to use Bash"}`), Want: with(agents.Event{Kind: agents.Notified, Message: "Claude needs your permission to use Bash"})},
		{Name: "Stop from the transcript", Payload: []byte(`{` + head + `"hook_event_name":"Stop","stop_hook_active":false}`), Want: with(agents.Event{Kind: agents.Stopped, Message: "PR #42 is ready\nThe refresh has its own module."})},
		{Name: "Stop with its message", Payload: []byte(`{` + head + `"hook_event_name":"Stop","last_assistant_message":"Done. Tests pass."}`), Want: with(agents.Event{Kind: agents.Stopped, Message: "Done. Tests pass."})},
		{Name: "SessionEnd", Payload: []byte(`{` + head + `"hook_event_name":"SessionEnd","reason":"prompt_input_exit"}`), Want: with(agents.Event{Kind: agents.Ended, Reason: "prompt_input_exit"})},
	}
}

func TestClaudeCodeKeepsTheHooksContract(t *testing.T) {
	contracttest.Hooks(t, func(t *testing.T) contracttest.HooksSubject {
		home := t.TempDir()
		settings := filepath.Join(home, ".claude", "settings.json")
		if err := os.MkdirAll(filepath.Dir(settings), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(settings, []byte(owners), 0o600); err != nil {
			t.Fatal(err)
		}
		transcript := filepath.Join(home, "transcript.jsonl")
		lines := `{"type":"user","message":{"role":"user","content":"Open the PR"}}` + "\n" +
			`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash"}]}}` + "\n" +
			`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"PR #42 is ready"},{"type":"text","text":"The refresh has its own module."}]}}` + "\n"
		if err := os.WriteFile(transcript, []byte(lines), 0o600); err != nil {
			t.Fatal(err)
		}
		return contracttest.HooksSubject{
			Hooks:      claudecode.New(local.New(nil, nil, nil, nil).FS, home, contracttest.Morning),
			OwnersKept: func(t *testing.T) bool { return ownersKept(t, settings) },
			Recorded:   recorded(transcript),
			Unknown: [][]byte{
				[]byte(`{"session_id":"s-1","hook_event_name":"PreCompact","trigger":"auto"}`),
				[]byte(`{"hook_event_name":"Stop"}`),
			},
		}
	})
}
