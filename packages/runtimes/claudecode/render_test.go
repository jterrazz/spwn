package claudecode

import (
	"encoding/json"
	"strings"
	"testing"

	"spwn.sh/packages/transpile"
)

// Renderer contract tests for claude-code. Whole-tree coverage lives
// under packages/runtimes/testdata/*/output_claude_code/; the cases
// here pin individual invariants so a break points at a specific
// section (settings.json shape, skill path convention, model pin)
// rather than a generic "goldens diverged".

func TestRenderer_Name(t *testing.T) {
	if got := Renderer.Name(); got != "claude-code" {
		t.Errorf("Name() = %q, want claude-code", got)
	}
}

// TestRender_BasePaths — every agent always gets CLAUDE.md, role.md,
// and a settings.json (the last one is owned by the renderer, not by
// spawn's DefaultConfigFiles, so we must see it here).
func TestRender_BasePaths(t *testing.T) {
	tree, err := Renderer.Render(transpile.Input{
		WorldID: "home",
		Agents:  []transpile.AgentInput{{Name: "neo", Role: "worker"}},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	want := []string{
		"agents/neo/CLAUDE.md",
		"agents/neo/worlds/home/role.md",
		"agents/neo/.claude/settings.json",
	}
	for _, p := range want {
		if !tree.Has(p) {
			t.Errorf("tree missing %s", p)
		}
	}
}

// TestRender_SettingsAlwaysSkipsDangerous — even the null input must
// emit a settings.json whose permissions flag is on; spwn containers
// are sandboxed and the prompt would block every one-shot invocation.
func TestRender_SettingsAlwaysSkipsDangerous(t *testing.T) {
	tree, _ := Renderer.Render(transpile.Input{
		WorldID: "home",
		Agents:  []transpile.AgentInput{{Name: "neo"}},
	})
	body, ok := tree.Get("agents/neo/.claude/settings.json")
	if !ok {
		t.Fatal("missing settings.json")
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("settings.json is not valid JSON: %v", err)
	}
	if parsed["skipDangerousModePermissionPrompt"] != true {
		t.Errorf("skipDangerousModePermissionPrompt = %v, want true", parsed["skipDangerousModePermissionPrompt"])
	}
}

// TestRender_ModelPinFlowsThrough — AgentInput.Model must land in
// settings.json so Claude Code auto-selects it at startup.
func TestRender_ModelPinFlowsThrough(t *testing.T) {
	tree, _ := Renderer.Render(transpile.Input{
		WorldID: "home",
		Agents:  []transpile.AgentInput{{Name: "neo", Model: "opus"}},
	})
	body, _ := tree.Get("agents/neo/.claude/settings.json")
	var parsed map[string]any
	_ = json.Unmarshal(body, &parsed)
	if parsed["model"] != "opus" {
		t.Errorf("model = %v, want opus", parsed["model"])
	}
}

// TestRender_EmptyModelOmitsKey — when no model override is set, we
// must not write `"model": ""` (Claude Code would treat that as
// "pin to the empty model"). Absence is correct.
func TestRender_EmptyModelOmitsKey(t *testing.T) {
	tree, _ := Renderer.Render(transpile.Input{
		WorldID: "home",
		Agents:  []transpile.AgentInput{{Name: "neo"}},
	})
	body, _ := tree.Get("agents/neo/.claude/settings.json")
	var parsed map[string]any
	_ = json.Unmarshal(body, &parsed)
	if _, has := parsed["model"]; has {
		t.Errorf("model key should be absent when no pin; got %v", parsed["model"])
	}
}

// TestRender_HooksFanIntoSettings — spwn/hooks.yaml entries must
// surface as Claude Code's nested {event: [{matcher, hooks:[{type,
// command}]}]} structure inside settings.json.
func TestRender_HooksFanIntoSettings(t *testing.T) {
	tree, _ := Renderer.Render(transpile.Input{
		WorldID: "home",
		Agents:  []transpile.AgentInput{{Name: "neo"}},
		Hooks: []transpile.HookEntry{
			{Name: "audit", Event: "PreToolUse", Matcher: "Bash", Command: "echo audit"},
			{Name: "welcome", Event: "SessionStart", Command: "echo hi"},
		},
	})
	body, _ := tree.Get("agents/neo/.claude/settings.json")
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
		t.Fatalf("settings.json hooks block malformed: %v\n%s", err, body)
	}
	pre := parsed.Hooks["PreToolUse"]
	if len(pre) != 1 {
		t.Fatalf("PreToolUse: got %d entries, want 1", len(pre))
	}
	if pre[0].Matcher != "Bash" {
		t.Errorf("PreToolUse matcher = %q, want Bash", pre[0].Matcher)
	}
	if len(pre[0].Hooks) != 1 || pre[0].Hooks[0].Type != "command" {
		t.Errorf("PreToolUse hook type: got %+v, want command", pre[0].Hooks)
	}
	if pre[0].Hooks[0].Command != "echo audit" {
		t.Errorf("PreToolUse command = %q, want \"echo audit\"", pre[0].Hooks[0].Command)
	}
	if _, has := parsed.Hooks["SessionStart"]; !has {
		t.Error("missing SessionStart event in hooks map")
	}
}

// TestRender_EmptyMatcherDefaultsToStar — every runtime interprets
// "*" as "match anything"; emitting an empty matcher would be a
// schema violation in both Claude Code and Codex.
func TestRender_EmptyMatcherDefaultsToStar(t *testing.T) {
	tree, _ := Renderer.Render(transpile.Input{
		WorldID: "home",
		Agents:  []transpile.AgentInput{{Name: "neo"}},
		Hooks: []transpile.HookEntry{
			{Name: "any", Event: "UserPromptSubmit", Command: "echo x"},
		},
	})
	body, _ := tree.Get("agents/neo/.claude/settings.json")
	if !strings.Contains(string(body), `"matcher": "*"`) {
		t.Errorf("empty matcher should default to \"*\"; got:\n%s", body)
	}
}

// TestRender_NoHooksOmitsHooksKey — absence is correct when no hooks
// are declared (keeps the file minimal + matches Claude Code's tolerant
// "hooks is optional" schema).
func TestRender_NoHooksOmitsHooksKey(t *testing.T) {
	tree, _ := Renderer.Render(transpile.Input{
		WorldID: "home",
		Agents:  []transpile.AgentInput{{Name: "neo"}},
	})
	body, _ := tree.Get("agents/neo/.claude/settings.json")
	var parsed map[string]any
	_ = json.Unmarshal(body, &parsed)
	if _, has := parsed["hooks"]; has {
		t.Errorf("hooks key should be absent when no hooks; got %v", parsed["hooks"])
	}
}

// TestRender_SkillsLandInClaudeSkillsTree — every skill emits every
// file it carries into `.claude/skills/<skill>/<relpath>` so the
// native walker finds both SKILL.md and any sidecar (templates,
// scripts).
func TestRender_SkillsLandInClaudeSkillsTree(t *testing.T) {
	tree, _ := Renderer.Render(transpile.Input{
		WorldID: "home",
		Agents:  []transpile.AgentInput{{Name: "neo"}},
		Skills: []transpile.SkillEntry{
			{
				Name: "greeter",
				Files: map[string][]byte{
					"SKILL.md":      []byte("---\nname: greeter\ndescription: Greet\n---\nHello."),
					"template.md":   []byte("Hi there."),
					"scripts/go.sh": []byte("#!/bin/bash\necho go"),
				},
			},
		},
	})
	for _, p := range []string{
		"agents/neo/.claude/skills/greeter/SKILL.md",
		"agents/neo/.claude/skills/greeter/template.md",
		"agents/neo/.claude/skills/greeter/scripts/go.sh",
	} {
		if !tree.Has(p) {
			t.Errorf("tree missing %s", p)
		}
	}
}

// TestRender_SkillsEmittedPerAgent — with multiple agents in a world,
// every agent gets its own copy of every skill. This is world-wide
// distribution: all agents see the same skill set, but the paths are
// per-agent so Claude Code's walker under each home works in
// isolation.
func TestRender_SkillsEmittedPerAgent(t *testing.T) {
	tree, _ := Renderer.Render(transpile.Input{
		WorldID: "colony",
		Agents: []transpile.AgentInput{
			{Name: "alice", Role: "chief"},
			{Name: "bob", Role: "worker"},
		},
		Skills: []transpile.SkillEntry{
			{Name: "shared", Files: map[string][]byte{"SKILL.md": []byte("---\nname: shared\ndescription: s\n---")}},
		},
	})
	for _, name := range []string{"alice", "bob"} {
		p := "agents/" + name + "/.claude/skills/shared/SKILL.md"
		if !tree.Has(p) {
			t.Errorf("tree missing %s — skills must fan out per agent", p)
		}
	}
}

// TestRender_NoSkillsLeavesNoSkillDir — a skill-less input must not
// emit a dangling `.claude/skills/` path. Absence is correct.
func TestRender_NoSkillsLeavesNoSkillDir(t *testing.T) {
	tree, _ := Renderer.Render(transpile.Input{
		WorldID: "home",
		Agents:  []transpile.AgentInput{{Name: "neo"}},
	})
	for _, p := range tree.Paths() {
		if strings.HasPrefix(p, "agents/neo/.claude/skills/") {
			t.Errorf("unexpected skill file without Skills input: %s", p)
		}
	}
}
