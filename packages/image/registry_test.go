package image

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
	err := r.Register(&stubTool{name: "@spwn/node", kind: KindSDK, version: "20"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Duplicate should fail
	err = r.Register(&stubTool{name: "@spwn/node", kind: KindSDK, version: "20"})
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "@spwn/node"})

	if r.Get("@spwn/node") == nil {
		t.Fatal("expected to find @spwn/node")
	}
	if r.Get("@missing") != nil {
		t.Fatal("expected nil for missing tool")
	}
}

func TestResolve_ExpandsDependencies(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "@spwn/node", version: "20"})
	r.Register(&stubTool{name: "@spwn/qmd", deps: []string{"@spwn/node"}})

	tools, err := r.Resolve([]string{"@spwn/qmd"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	// @spwn/node must come before @spwn/qmd
	if tools[0].Name() != "@spwn/node" || tools[1].Name() != "@spwn/qmd" {
		t.Fatalf("wrong order: %s, %s", tools[0].Name(), tools[1].Name())
	}
}

func TestResolve_Deduplicates(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "@spwn/node"})
	r.Register(&stubTool{name: "@spwn/qmd", deps: []string{"@spwn/node"}})

	// Request both @spwn/node and @spwn/qmd - @spwn/node should appear only once
	tools, err := r.Resolve([]string{"@spwn/node", "@spwn/qmd"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools (deduplicated), got %d", len(tools))
	}
}

func TestResolve_TopologicalOrder(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "@spwn/unix"})
	r.Register(&stubTool{name: "@spwn/node", deps: []string{"@spwn/unix"}})
	r.Register(&stubTool{name: "@spwn/claude-code", deps: []string{"@spwn/node"}})
	r.Register(&stubTool{name: "@spwn/qmd", deps: []string{"@spwn/node"}})

	tools, err := r.Resolve([]string{"@spwn/qmd", "@spwn/claude-code"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Build index map for order checking
	idx := make(map[string]int)
	for i, t := range tools {
		idx[t.Name()] = i
	}

	if idx["@spwn/unix"] >= idx["@spwn/node"] {
		t.Error("@spwn/unix should come before @spwn/node")
	}
	if idx["@spwn/node"] >= idx["@spwn/claude-code"] {
		t.Error("@spwn/node should come before @spwn/claude-code")
	}
	if idx["@spwn/node"] >= idx["@spwn/qmd"] {
		t.Error("@spwn/node should come before @spwn/qmd")
	}
}

func TestResolve_MissingTool(t *testing.T) {
	r := NewRegistry()
	_, err := r.Resolve([]string{"@spwn/nonexistent"})
	if err == nil {
		t.Fatal("expected error for missing tool")
	}
}

func TestResolve_MissingDependency(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "@spwn/qmd", deps: []string{"@spwn/node"}})

	_, err := r.Resolve([]string{"@spwn/qmd"})
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}

func TestResolve_CycleDetection(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "@spwn/a", deps: []string{"@spwn/b"}})
	r.Register(&stubTool{name: "@spwn/b", deps: []string{"@spwn/a"}})

	_, err := r.Resolve([]string{"@spwn/a"})
	if err == nil {
		t.Fatal("expected error for dependency cycle")
	}
}

func TestResolve_NoDeps(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "@spwn/unix"})
	r.Register(&stubTool{name: "@spwn/git"})

	tools, err := r.Resolve([]string{"@spwn/unix", "@spwn/git"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
}
