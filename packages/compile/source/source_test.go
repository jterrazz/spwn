package source

import (
	"path/filepath"
	"testing"
)

func TestLoadMinimalProject(t *testing.T) {
	src, err := Load(filepath.Join("testdata", "minimal"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if src.Manifest == nil {
		t.Fatal("nil manifest")
	}
	if src.Manifest.Name != "minimal" {
		t.Errorf("project name: got %q, want %q", src.Manifest.Name, "minimal")
	}
	if len(src.Agents) != 1 {
		t.Fatalf("agents: got %d, want 1", len(src.Agents))
	}
	neo := src.Agents[0]
	if neo.Name != "neo" {
		t.Errorf("agent name: got %q", neo.Name)
	}
	if len(neo.AgentMD) == 0 {
		t.Error("AgentMD is empty")
	}
	if neo.Config.Name != "neo" {
		t.Errorf("agent.yaml name: got %q", neo.Config.Name)
	}
	if neo.Config.Role != "worker" {
		t.Errorf("agent.yaml role: got %q", neo.Config.Role)
	}
	if len(neo.Config.Tools) != 2 {
		t.Errorf("agent.yaml tools: got %d, want 2", len(neo.Config.Tools))
	}
	if got, want := len(neo.Layers.Skills), 1; got != want {
		t.Errorf("layer skills: got %d, want %d", got, want)
	}
	if _, ok := neo.Layers.Skills["warm-up.md"]; !ok {
		t.Errorf("skills layer missing warm-up.md; keys: %v", keys(neo.Layers.Skills))
	}
	if len(src.Skills) != 1 {
		t.Errorf("skills: got %d, want 1", len(src.Skills))
	}
	if src.Skills[0].Name != "shared-skill" {
		t.Errorf("skill name: got %q, want %q", src.Skills[0].Name, "shared-skill")
	}
	if len(src.Hooks) != 1 {
		t.Errorf("hooks: got %d, want 1", len(src.Hooks))
	}
	if src.Hooks[0].Name != "pre-commit" {
		t.Errorf("hook name: got %q", src.Hooks[0].Name)
	}
}

func TestLoadMissingManifest(t *testing.T) {
	_, err := Load(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing spwn.yaml")
	}
}

func TestToCompileInputPicksSoleWorld(t *testing.T) {
	src, err := Load(filepath.Join("testdata", "minimal"))
	if err != nil {
		t.Fatal(err)
	}
	in, err := ToCompileInput(src, "")
	if err != nil {
		t.Fatalf("ToCompileInput: %v", err)
	}
	if in.WorldID != "home" {
		t.Errorf("WorldID: got %q, want %q", in.WorldID, "home")
	}
	if len(in.Agents) != 1 || in.Agents[0].Name != "neo" {
		t.Errorf("agents: got %v", in.Agents)
	}
	if in.Agents[0].Role != "worker" {
		t.Errorf("role: got %q", in.Agents[0].Role)
	}
	wantTools := []string{"@spwn/git", "@spwn/unix"}
	if got := in.VerifiedTools; !equalStrings(got, wantTools) {
		t.Errorf("tools: got %v, want %v", got, wantTools)
	}
}

func TestToCompileInputMissingWorld(t *testing.T) {
	src, err := Load(filepath.Join("testdata", "minimal"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = ToCompileInput(src, "bogus")
	if err == nil {
		t.Fatal("expected error for unknown world")
	}
}

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
