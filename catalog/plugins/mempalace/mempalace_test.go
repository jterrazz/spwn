package mempalace

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"testing"

	ib "spwn.sh/packages/image"
)

func TestPlugin_Identity(t *testing.T) {
	if got := Tool.Name(); got != "@spwn/mempalace" {
		t.Errorf("Name = %q, want @spwn/mempalace", got)
	}
	if got := Tool.Kind(); got != ib.KindTool {
		t.Errorf("Kind = %q, want tool", got)
	}
	if got := Tool.Version(); got == "" {
		t.Errorf("Version = %q, want non-empty", got)
	}
}

func TestPlugin_DependsOnPython(t *testing.T) {
	deps := Tool.Dependencies()
	var found bool
	for _, d := range deps {
		if d == "@spwn/python" {
			found = true
		}
	}
	if !found {
		t.Errorf("Dependencies = %v, want to include @spwn/python", deps)
	}
}

func TestPlugin_InstallSpecNotEmpty(t *testing.T) {
	spec := Tool.Install()
	if len(spec.Commands) == 0 {
		t.Error("Install().Commands is empty; expected at least the pip install line")
	}
}

func TestPlugin_VerifyNotEmpty(t *testing.T) {
	if len(Tool.Verify()) == 0 {
		t.Error("Verify() is empty")
	}
}

func TestPlugin_TargetsClaudeCode(t *testing.T) {
	rts := ib.PluginRuntimes(Tool)
	if len(rts) == 0 {
		t.Fatal("PluginRuntimes empty; plugin should target at least one runtime")
	}
	var found bool
	for _, r := range rts {
		if r == "@spwn/claude-code" {
			found = true
		}
	}
	if !found {
		t.Errorf("Runtimes = %v, want to include @spwn/claude-code", rts)
	}
}

func TestPlugin_ConfigShape(t *testing.T) {
	cfg := ib.PluginConfig(Tool, "@spwn/claude-code")
	if len(cfg) == 0 {
		t.Fatal("PluginConfig(claude-code) returned no bytes")
	}
	var m map[string]any
	if err := json.Unmarshal(cfg, &m); err != nil {
		t.Fatalf("config is not valid JSON: %v\n%s", err, cfg)
	}
	mcp, ok := m["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("config missing mcpServers object: %s", cfg)
	}
	if _, ok := mcp["mempalace"]; !ok {
		t.Errorf("mcpServers missing `mempalace` key: %v", mcp)
	}
}

func TestPlugin_ConfigGatedByRuntime(t *testing.T) {
	if got := ib.PluginConfig(Tool, "@spwn/codex"); got != nil {
		t.Errorf("Config(codex) = %q, want nil (not a declared runtime)", got)
	}
}

func TestPlugin_SkillsContainsSkillMD(t *testing.T) {
	skills := Tool.Skills()
	if skills == nil {
		t.Fatal("Skills() returned nil")
	}
	data, err := fs.ReadFile(skills, "SKILL.md")
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	if !bytes.Contains(data, []byte("mempalace")) {
		t.Error("SKILL.md does not mention mempalace")
	}
}
