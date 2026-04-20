package codex

import (
	"strings"
	"testing"

	"spwn.sh/packages/runtimes"
)

// Codex one-shot contract — lives separately from spawn_test.go so
// the argv + JSONL-parsing shape is readable on its own. See
// apps/cli/agent/talk.go for how these are chained at runtime.

func TestBuildCommand_oneShot(t *testing.T) {
	got := Spawner.BuildCommand(runtimes.SpawnConfig{
		AgentName: "neo",
		WorldID:   "w-1",
		Prompt:    "hello",
	})
	assertEqStrings(t, got, []string{"codex", "exec", "hello"})
}

func TestBuildCommand_oneShotResume(t *testing.T) {
	got := Spawner.BuildCommand(runtimes.SpawnConfig{
		AgentName: "neo",
		WorldID:   "w-1",
		Prompt:    "follow up",
		SessionID: "th_abc",
	})
	assertEqStrings(t, got, []string{"codex", "exec", "--thread", "th_abc", "follow up"})
}

func TestBuildCommand_namedAgentNoPromptIsInteractive(t *testing.T) {
	// Named agent without a prompt is how the architect spawns an
	// agent in detached mode — it wants a blocking REPL process
	// running inside the container. That's the same shape as the
	// anonymous interactive case: `codex` with no flags.
	got := Spawner.BuildCommand(runtimes.SpawnConfig{AgentName: "neo"})
	assertEqStrings(t, got, []string{"codex"})
}

func TestOneShotFlags_appendsJSON(t *testing.T) {
	base := []string{"codex", "exec", "hi"}
	got := Spawner.OneShotFlags(base, "")
	assertEqStrings(t, got, []string{"codex", "exec", "hi", "--json"})
}

func TestOneShotFlags_streamFormat(t *testing.T) {
	// Codex has no separate stream-vs-envelope format — `--json` is
	// JSONL regardless. stream-json from the caller resolves to the
	// same flag set; the difference lives in how talk.go consumes
	// stdout, not in codex's CLI.
	base := []string{"codex", "exec", "hi"}
	got := Spawner.OneShotFlags(base, "stream-json")
	assertEqStrings(t, got, []string{"codex", "exec", "hi", "--json"})
}

func TestOneShotFlags_emptyBaseNoOp(t *testing.T) {
	// BuildCommand can return nil on degenerate config; don't
	// promote that to a runnable argv by sticking `--json` onto it.
	got := Spawner.OneShotFlags(nil, "")
	if got != nil {
		t.Errorf("expected nil for empty base, got %v", got)
	}
}

func TestParseOneShotResult_happy(t *testing.T) {
	raw := []byte(strings.Join([]string{
		`{"type":"thread.started","thread_id":"th_abc"}`,
		`{"type":"turn.started","turn_id":"t_1"}`,
		`{"type":"item.completed","item":{"item_type":"agent_message","text":"hello from codex"}}`,
		`{"type":"turn.completed"}`,
	}, "\n"))
	text, sid, err := Spawner.ParseOneShotResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "hello from codex" {
		t.Errorf("text = %q", text)
	}
	if sid != "th_abc" {
		t.Errorf("sessionID = %q, want th_abc", sid)
	}
}

func TestParseOneShotResult_multiTurnLastWins(t *testing.T) {
	// A single prompt can produce multiple agent_message items; the
	// LAST is the final reply. Earlier ones are intermediate drafts.
	raw := []byte(strings.Join([]string{
		`{"type":"thread.started","thread_id":"th_xyz"}`,
		`{"type":"item.completed","item":{"item_type":"agent_message","text":"first draft"}}`,
		`{"type":"item.completed","item":{"item_type":"reasoning","text":"thought"}}`,
		`{"type":"item.completed","item":{"item_type":"agent_message","text":"final answer"}}`,
		`{"type":"turn.completed"}`,
	}, "\n"))
	text, _, err := Spawner.ParseOneShotResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "final answer" {
		t.Errorf("expected the LAST agent_message, got %q", text)
	}
}

func TestParseOneShotResult_nonAgentItemsIgnored(t *testing.T) {
	// Reasoning and tool-execution items surface inside the stream
	// alongside agent replies — they're not "the answer" and must
	// be skipped.
	raw := []byte(strings.Join([]string{
		`{"type":"thread.started","thread_id":"th_abc"}`,
		`{"type":"item.completed","item":{"item_type":"reasoning","text":"thinking..."}}`,
		`{"type":"item.completed","item":{"item_type":"command_execution","command":"ls"}}`,
		`{"type":"item.completed","item":{"item_type":"agent_message","text":"done"}}`,
	}, "\n"))
	text, _, err := Spawner.ParseOneShotResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "done" {
		t.Errorf("text = %q, want done", text)
	}
}

func TestParseOneShotResult_emptyInputErrors(t *testing.T) {
	_, _, err := Spawner.ParseOneShotResult([]byte(""))
	if err == nil {
		t.Fatal("expected error on empty input")
	}
}

func TestParseOneShotResult_noJSONErrors(t *testing.T) {
	_, _, err := Spawner.ParseOneShotResult([]byte("plain text output"))
	if err == nil {
		t.Fatal("expected error when no JSON events present")
	}
}

func TestParseOneShotResult_malformedLinesIgnored(t *testing.T) {
	// A warning on stderr that somehow snuck into stdout, a truncated
	// final line, a stray blank line — none of these should stop us
	// extracting what IS valid.
	raw := []byte(strings.Join([]string{
		`warning: this is not json`,
		`{"type":"thread.started","thread_id":"th_abc"}`,
		`{truncated at end`,
		`{"type":"item.completed","item":{"item_type":"agent_message","text":"ok"}}`,
	}, "\n"))
	text, sid, err := Spawner.ParseOneShotResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "ok" || sid != "th_abc" {
		t.Errorf("text=%q sid=%q", text, sid)
	}
}

func assertEqStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %d %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}
