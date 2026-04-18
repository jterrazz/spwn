package dependency_test

import (
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/dependency"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in    string
		kind  dependency.RefKind
		owner string
		name  string
	}{
		{"spwn:unix", dependency.KindSpwnBuiltin, "spwn", "unix"},
		{"spwn:claude-code", dependency.KindSpwnBuiltin, "spwn", "claude-code"},
		{"github:acme/foo", dependency.KindRegistry, "acme", "foo"},
		{"github:jterrazz/python", dependency.KindRegistry, "jterrazz", "python"},
		{"skill:focus", dependency.KindLocalSkill, "", "focus"},
		{"tool:my-parser", dependency.KindLocalTool, "", "my-parser"},
		{"hook:pre-spawn", dependency.KindLocalHook, "", "pre-spawn"},
		// Bare names are now invalid under the new grammar.
		{"local-tool", dependency.KindInvalid, "", ""},
		{"  spaced  ", dependency.KindInvalid, "", ""},
		// Legacy `@owner/name` form is malformed.
		{"@acme/foo", dependency.KindInvalid, "", ""},
		{"@jterrazz/python", dependency.KindInvalid, "", ""},
		{"@malformed", dependency.KindInvalid, "", ""},
		// The legacy `local:<name>` alias was retired alongside bare names.
		{"local:my-parser", dependency.KindInvalid, "", ""},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := dependency.ParseRef(c.in)
			if got.Kind != c.kind {
				t.Errorf("kind: want %v, got %v", c.kind, got.Kind)
			}
			if got.Owner != c.owner {
				t.Errorf("owner: want %q, got %q", c.owner, got.Owner)
			}
			if got.Name != c.name {
				t.Errorf("name: want %q, got %q", c.name, got.Name)
			}
		})
	}
}

func TestSplitVersion(t *testing.T) {
	cases := []struct {
		in         string
		dependency string
		version    string
	}{
		{"spwn:unix", "spwn:unix", ""},
		{"spwn:unix@24.04", "spwn:unix", "24.04"},
		{"skill:focus", "skill:focus", ""},
		{"tool:my-parser@0.1", "tool:my-parser", "0.1"},
		{"github:acme/foo@1.2.3", "github:acme/foo", "1.2.3"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			dependency, version := dependency.SplitVersion(c.in)
			if dependency != c.dependency {
				t.Errorf("dependency: want %q, got %q", c.dependency, dependency)
			}
			if version != c.version {
				t.Errorf("version: want %q, got %q", c.version, version)
			}
		})
	}
}

func TestResolveTool_LocalTool(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "spwn", "tools", "present"))

	got := dependency.ResolveTool(root, dependency.ParseRef("tool:present"), nil, false)
	if got != dependency.ResolveOK {
		t.Errorf("present local tool: want OK, got %v", got)
	}

	got = dependency.ResolveTool(root, dependency.ParseRef("tool:missing"), nil, false)
	if got != dependency.ResolveNotFound {
		t.Errorf("missing local tool: want NotFound, got %v", got)
	}
}

func TestResolveTool_LocalSkill(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "spwn", "skills"))
	if err := os.WriteFile(filepath.Join(root, "spwn", "skills", "focus.md"), []byte("# focus"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := dependency.ResolveTool(root, dependency.ParseRef("skill:focus"), nil, false)
	if got != dependency.ResolveOK {
		t.Errorf("present local skill: want OK, got %v", got)
	}

	got = dependency.ResolveTool(root, dependency.ParseRef("skill:missing"), nil, false)
	if got != dependency.ResolveNotFound {
		t.Errorf("missing local skill: want NotFound, got %v", got)
	}
}

func TestResolveTool_LocalHook(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "spwn", "hooks"))
	if err := os.WriteFile(filepath.Join(root, "spwn", "hooks", "pre-spawn.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := dependency.ResolveTool(root, dependency.ParseRef("hook:pre-spawn"), nil, false)
	if got != dependency.ResolveOK {
		t.Errorf("present local hook: want OK, got %v", got)
	}

	got = dependency.ResolveTool(root, dependency.ParseRef("hook:missing"), nil, false)
	if got != dependency.ResolveNotFound {
		t.Errorf("missing local hook: want NotFound, got %v", got)
	}
}

func TestResolveTool_InvalidBareName(t *testing.T) {
	root := t.TempDir()
	// Even if a file named "present" existed in spwn/tools/, a bare
	// ref must NOT resolve — the new grammar rejects it up front.
	mustMkdir(t, filepath.Join(root, "spwn", "tools", "present"))

	got := dependency.ResolveTool(root, dependency.ParseRef("present"), nil, false)
	if got != dependency.ResolveInvalid {
		t.Errorf("bare ref: want ResolveInvalid, got %v", got)
	}
}

func TestResolveTool_Builtin(t *testing.T) {
	builtin := map[string]struct{}{
		"spwn:unix": {},
		"spwn:git":  {},
	}

	got := dependency.ResolveTool("", dependency.ParseRef("spwn:unix"), builtin, true)
	if got != dependency.ResolveOK {
		t.Errorf("known builtin: want OK, got %v", got)
	}

	got = dependency.ResolveTool("", dependency.ParseRef("spwn:nonesuch"), builtin, true)
	if got != dependency.ResolveNotFound {
		t.Errorf("unknown builtin with catalog: want NotFound, got %v", got)
	}

	got = dependency.ResolveTool("", dependency.ParseRef("spwn:nonesuch"), nil, false)
	if got != dependency.ResolveOK {
		t.Errorf("unknown builtin without catalog: want OK (permissive), got %v", got)
	}
}

func TestResolveTool_Registry(t *testing.T) {
	got := dependency.ResolveTool("", dependency.ParseRef("github:acme/foo"), nil, false)
	if got != dependency.ResolveRegistryUnsupported {
		t.Errorf("registry ref: want RegistryUnsupported, got %v", got)
	}
}

func TestResolveSkill_SchemeForm(t *testing.T) {
	root := t.TempDir()

	// skill: scheme — file form.
	mustMkdir(t, filepath.Join(root, "spwn", "skills"))
	if err := os.WriteFile(filepath.Join(root, "spwn", "skills", "focus.md"), []byte("# focus"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := dependency.ResolveSkill(root, dependency.ParseRef("skill:focus"), nil, false)
	if got != dependency.ResolveOK {
		t.Errorf("skill: scheme resolves: want OK, got %v", got)
	}

	// tool: scheme — directory form.
	mustMkdir(t, filepath.Join(root, "spwn", "tools", "debug"))
	got = dependency.ResolveSkill(root, dependency.ParseRef("tool:debug"), nil, false)
	if got != dependency.ResolveOK {
		t.Errorf("tool: scheme resolves: want OK, got %v", got)
	}

	// Bare name is rejected outright.
	got = dependency.ResolveSkill(root, dependency.ParseRef("focus"), nil, false)
	if got != dependency.ResolveInvalid {
		t.Errorf("bare ref via ResolveSkill: want ResolveInvalid, got %v", got)
	}

	got = dependency.ResolveSkill(root, dependency.ParseRef("skill:missing"), nil, false)
	if got != dependency.ResolveNotFound {
		t.Errorf("missing skill: scheme: want NotFound, got %v", got)
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}
