package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// claudeEvents and codexEvents mirror packages/runtimes/<runtime>/events.go
// without importing those packages — depguard forbids the validate
// (project layer) from depending on the runtimes (build layer). Keep
// them in sync; if the runtime sets ever drift far enough that a test
// drift becomes likely, hoist a shared registry into a layer below
// both packages instead.
var (
	claudeEvents = []string{
		"Notification", "PostToolUse", "PreCompact", "PreToolUse",
		"SessionEnd", "SessionStart", "Stop", "SubagentStop", "UserPromptSubmit",
	}
	codexEvents = []string{
		"PostToolUse", "PreToolUse", "SessionStart", "Stop", "UserPromptSubmit",
	}
	dualRuntimeEvents = map[string][]string{
		"claude-code": claudeEvents,
		"codex":       codexEvents,
	}
)

func writeHook(t *testing.T, root, name, body string) {
	t.Helper()
	dir := filepath.Join(root, "spwn", "hooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write hook: %v", err)
	}
}

// TestHookEvents_supportedEventClean: a hook on PreToolUse selected by
// a claude-code agent must produce zero issues.
func TestHookEvents_supportedEventClean(t *testing.T) {
	root := t.TempDir()
	writeHook(t, root, "audit", "event: PreToolUse\ncommand: echo hi\n")
	ref := scaffoldAgent(t, root, "alpha", `name: alpha
runtime:
  backend: spwn:claude-code
dependencies:
  - hook/audit
`)
	in := minimalInput(root, []AgentRef{ref}, nil)
	in.HookEventsByRuntime = dualRuntimeEvents
	issues := ruleHookEventsSupported(in)
	if len(issues) != 0 {
		t.Errorf("supported event should not warn, got: %+v", issues)
	}
}

// TestHookEvents_unsupportedEventOnAgentRuntime: a hook on a Claude-only
// event selected by a codex agent must surface a LevelWarning naming
// the agent + runtime + event.
func TestHookEvents_unsupportedEventOnAgentRuntime(t *testing.T) {
	root := t.TempDir()
	writeHook(t, root, "compact", "event: PreCompact\ncommand: echo bye\n")
	ref := scaffoldAgent(t, root, "alpha", `name: alpha
runtime:
  backend: spwn:codex
dependencies:
  - hook/compact
`)
	in := minimalInput(root, []AgentRef{ref}, nil)
	in.HookEventsByRuntime = dualRuntimeEvents
	issues := ruleHookEventsSupported(in)
	if len(issues) != 1 {
		t.Fatalf("want 1 warning, got %d: %+v", len(issues), issues)
	}
	if issues[0].Level != LevelWarning {
		t.Errorf("level: got %v, want LevelWarning", issues[0].Level)
	}
	for _, want := range []string{"PreCompact", "codex", "alpha"} {
		if !strings.Contains(issues[0].Message, want) {
			t.Errorf("message missing %q: %s", want, issues[0].Message)
		}
	}
}

// TestHookEvents_orphanHookFile: a hook file that no agent subscribes
// to must surface a LevelInfo so dead authoring doesn't accumulate.
func TestHookEvents_orphanHookFile(t *testing.T) {
	root := t.TempDir()
	writeHook(t, root, "lonely", "event: PreToolUse\ncommand: echo none\n")
	ref := scaffoldAgent(t, root, "alpha", `name: alpha
runtime:
  backend: spwn:claude-code
dependencies: []
`)
	in := minimalInput(root, []AgentRef{ref}, nil)
	in.HookEventsByRuntime = dualRuntimeEvents
	issues := ruleHookEventsSupported(in)
	if len(issues) != 1 {
		t.Fatalf("want 1 info, got %d: %+v", len(issues), issues)
	}
	if issues[0].Level != LevelInfo {
		t.Errorf("level: got %v, want LevelInfo", issues[0].Level)
	}
	if !strings.Contains(issues[0].Message, "lonely") {
		t.Errorf("message missing hook name: %s", issues[0].Message)
	}
}

// TestHookEvents_emptyRegistrySkips: when the caller doesn't pass a
// HookEventsByRuntime map, the rule must no-op so golden tests and
// scaffold paths that lack runtime context stay quiet.
func TestHookEvents_emptyRegistrySkips(t *testing.T) {
	root := t.TempDir()
	writeHook(t, root, "wild", "event: TotallyMadeUp\ncommand: echo\n")
	ref := scaffoldAgent(t, root, "alpha", `name: alpha
runtime:
  backend: spwn:codex
dependencies:
  - hook/wild
`)
	in := minimalInput(root, []AgentRef{ref}, nil)
	// in.HookEventsByRuntime intentionally left nil
	if issues := ruleHookEventsSupported(in); len(issues) != 0 {
		t.Errorf("nil registry must skip, got %+v", issues)
	}
}

// TestHookEvents_projectDefaultRuntime: an agent with no runtime.backend
// inherits the project-level spwn.yaml#runtime.backend; the rule must
// resolve through that fallback when classifying events.
func TestHookEvents_projectDefaultRuntime(t *testing.T) {
	root := t.TempDir()
	writeHook(t, root, "compact", "event: PreCompact\ncommand: echo\n")
	ref := scaffoldAgent(t, root, "alpha", `name: alpha
dependencies:
  - hook/compact
`)
	in := minimalInput(root, []AgentRef{ref}, nil)
	in.Manifest.Runtime.Backend = "spwn:codex"
	in.HookEventsByRuntime = dualRuntimeEvents
	issues := ruleHookEventsSupported(in)
	if len(issues) != 1 {
		t.Fatalf("want 1 warning via project-default runtime, got %d: %+v", len(issues), issues)
	}
	if !strings.Contains(issues[0].Message, "codex") {
		t.Errorf("expected runtime in message, got: %s", issues[0].Message)
	}
}
