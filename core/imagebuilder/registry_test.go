package imagebuilder

import (
	"io/fs"
	"testing"
)

// stubTool is a minimal Tool implementation for testing.
type stubTool struct {
	name    string
	kind    Kind
	version string
	deps    []string
	install InstallSpec
	verify  []string
	skills  fs.FS
}

func (s *stubTool) Name() string         { return s.name }
func (s *stubTool) Kind() Kind           { return s.kind }
func (s *stubTool) Version() string      { return s.version }
func (s *stubTool) Dependencies() []string { return s.deps }
func (s *stubTool) Install() InstallSpec { return s.install }
func (s *stubTool) Verify() []string     { return s.verify }
func (s *stubTool) Skills() fs.FS        { return s.skills }

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	err := r.Register(&stubTool{name: "@node", kind: KindSDK, version: "20"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Duplicate should fail
	err = r.Register(&stubTool{name: "@node", kind: KindSDK, version: "20"})
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "@node"})

	if r.Get("@node") == nil {
		t.Fatal("expected to find @node")
	}
	if r.Get("@missing") != nil {
		t.Fatal("expected nil for missing tool")
	}
}

func TestResolve_ExpandsDependencies(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "@node", version: "20"})
	r.Register(&stubTool{name: "@qmd", deps: []string{"@node"}})

	tools, err := r.Resolve([]string{"@qmd"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	// @node must come before @qmd
	if tools[0].Name() != "@node" || tools[1].Name() != "@qmd" {
		t.Fatalf("wrong order: %s, %s", tools[0].Name(), tools[1].Name())
	}
}

func TestResolve_Deduplicates(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "@node"})
	r.Register(&stubTool{name: "@qmd", deps: []string{"@node"}})

	// Request both @node and @qmd — @node should appear only once
	tools, err := r.Resolve([]string{"@node", "@qmd"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools (deduplicated), got %d", len(tools))
	}
}

func TestResolve_TopologicalOrder(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "@unix"})
	r.Register(&stubTool{name: "@node", deps: []string{"@unix"}})
	r.Register(&stubTool{name: "@claude-code", deps: []string{"@node"}})
	r.Register(&stubTool{name: "@qmd", deps: []string{"@node"}})

	tools, err := r.Resolve([]string{"@qmd", "@claude-code"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Build index map for order checking
	idx := make(map[string]int)
	for i, t := range tools {
		idx[t.Name()] = i
	}

	if idx["@unix"] >= idx["@node"] {
		t.Error("@unix should come before @node")
	}
	if idx["@node"] >= idx["@claude-code"] {
		t.Error("@node should come before @claude-code")
	}
	if idx["@node"] >= idx["@qmd"] {
		t.Error("@node should come before @qmd")
	}
}

func TestResolve_MissingTool(t *testing.T) {
	r := NewRegistry()
	_, err := r.Resolve([]string{"@nonexistent"})
	if err == nil {
		t.Fatal("expected error for missing tool")
	}
}

func TestResolve_MissingDependency(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "@qmd", deps: []string{"@node"}})

	_, err := r.Resolve([]string{"@qmd"})
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}

func TestResolve_CycleDetection(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "@a", deps: []string{"@b"}})
	r.Register(&stubTool{name: "@b", deps: []string{"@a"}})

	_, err := r.Resolve([]string{"@a"})
	if err == nil {
		t.Fatal("expected error for dependency cycle")
	}
}

func TestResolve_NoDeps(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "@unix"})
	r.Register(&stubTool{name: "@git"})

	tools, err := r.Resolve([]string{"@unix", "@git"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
}
