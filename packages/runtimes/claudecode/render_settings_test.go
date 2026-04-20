package claudecode

import (
	"encoding/json"
	"strings"
	"testing"

	"spwn.sh/packages/transpile"
)

// Unit tests for GenerateAgentSettingsJSON. Renderer-level tests in
// render_test.go exercise the happy path; these zoom in on the JSON
// shape invariants a runtime will break on if we drift (key order is
// encoded in json.Marshal's map-sorting behaviour; a divergence there
// would surface as a golden diff elsewhere).

func TestGenerateAgentSettingsJSON_alwaysEmitsPermissionFlag(t *testing.T) {
	body := GenerateAgentSettingsJSON(SettingsInput{})
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, body)
	}
	if parsed["skipDangerousModePermissionPrompt"] != true {
		t.Error("skipDangerousModePermissionPrompt must always be true")
	}
}

func TestGenerateAgentSettingsJSON_omitsModelWhenEmpty(t *testing.T) {
	body := GenerateAgentSettingsJSON(SettingsInput{})
	if strings.Contains(string(body), `"model"`) {
		t.Errorf("empty Model should not write the key; got:\n%s", body)
	}
}

func TestGenerateAgentSettingsJSON_includesModel(t *testing.T) {
	body := GenerateAgentSettingsJSON(SettingsInput{Model: "sonnet"})
	if !strings.Contains(string(body), `"model": "sonnet"`) {
		t.Errorf("expected model pin; got:\n%s", body)
	}
}

func TestGenerateAgentSettingsJSON_omitsHooksWhenEmpty(t *testing.T) {
	body := GenerateAgentSettingsJSON(SettingsInput{})
	if strings.Contains(string(body), `"hooks"`) {
		t.Errorf("empty Hooks should not write the key; got:\n%s", body)
	}
}

// TestGenerateAgentSettingsJSON_groupsHooksByEvent — two hooks on the
// same event must collapse into the SAME event's array (not produce
// two competing PreToolUse keys, which isn't valid JSON anyway).
func TestGenerateAgentSettingsJSON_groupsHooksByEvent(t *testing.T) {
	body := GenerateAgentSettingsJSON(SettingsInput{
		Hooks: []transpile.HookEntry{
			{Name: "a", Event: "PreToolUse", Matcher: "Bash", Command: "a"},
			{Name: "b", Event: "PreToolUse", Matcher: "Edit", Command: "b"},
			{Name: "c", Event: "SessionStart", Command: "c"},
		},
	})
	var parsed struct {
		Hooks map[string][]any `json:"hooks"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("bad JSON: %v\n%s", err, body)
	}
	if len(parsed.Hooks) != 2 {
		t.Fatalf("events: got %d, want 2 (PreToolUse + SessionStart)", len(parsed.Hooks))
	}
	if len(parsed.Hooks["PreToolUse"]) != 2 {
		t.Errorf("PreToolUse entries: got %d, want 2", len(parsed.Hooks["PreToolUse"]))
	}
	if len(parsed.Hooks["SessionStart"]) != 1 {
		t.Errorf("SessionStart entries: got %d, want 1", len(parsed.Hooks["SessionStart"]))
	}
}

// TestGenerateAgentSettingsJSON_hookShapeMatchesClaudeSchema — the
// inner envelope must be `{matcher, hooks:[{type:"command", command}]}`
// per https://code.claude.com/docs/en/hooks.md. Drift here silently
// breaks claude at runtime.
func TestGenerateAgentSettingsJSON_hookShapeMatchesClaudeSchema(t *testing.T) {
	body := GenerateAgentSettingsJSON(SettingsInput{
		Hooks: []transpile.HookEntry{
			{Name: "x", Event: "PreToolUse", Matcher: "Bash", Command: "echo"},
		},
	})
	var parsed struct {
		Hooks map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	entry := parsed.Hooks["PreToolUse"][0]
	if entry.Matcher != "Bash" || entry.Hooks[0].Type != "command" || entry.Hooks[0].Command != "echo" {
		t.Errorf("hook envelope shape drift: %+v", entry)
	}
}

func TestGenerateAgentSettingsJSON_emptyMatcherBecomesStar(t *testing.T) {
	body := GenerateAgentSettingsJSON(SettingsInput{
		Hooks: []transpile.HookEntry{
			{Name: "any", Event: "SessionStart", Command: "c"},
		},
	})
	if !strings.Contains(string(body), `"matcher": "*"`) {
		t.Errorf("empty matcher must default to \"*\"; got:\n%s", body)
	}
}

// TestGenerateAgentSettingsJSON_isValidJSONWithTrailingNewline — the
// helper appends a trailing newline so `cat` output is POSIX-clean
// and the file diffs cleanly against goldens.
func TestGenerateAgentSettingsJSON_isValidJSONWithTrailingNewline(t *testing.T) {
	body := GenerateAgentSettingsJSON(SettingsInput{})
	if len(body) == 0 || body[len(body)-1] != '\n' {
		t.Error("settings.json body must end with a newline")
	}
	if !json.Valid(body) {
		t.Errorf("body is not valid JSON: %s", body)
	}
}
