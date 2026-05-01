package refs_test

import (
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/dependency/refs"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in    string
		kind  refs.RefKind
		owner string
		name  string
	}{
		{"spwn:unix", refs.KindSpwnBuiltin, "spwn", "unix"},
		{"spwn:claude-code", refs.KindSpwnBuiltin, "spwn", "claude-code"},
		{"github:acme/foo", refs.KindRegistry, "acme", "foo"},
		{"github:jterrazz/python", refs.KindRegistry, "jterrazz", "python"},
		{"skill/focus", refs.KindLocalSkill, "", "focus"},
		{"tool/my-parser", refs.KindLocalTool, "", "my-parser"},
		{"hook/pre-spawn", refs.KindLocalHook, "", "pre-spawn"},
		// Bare names are invalid.
		{"local-tool", refs.KindInvalid, "", ""},
		{"  spaced  ", refs.KindInvalid, "", ""},
		// Legacy `@owner/name` form is malformed.
		{"@acme/foo", refs.KindInvalid, "", ""},
		{"@jterrazz/python", refs.KindInvalid, "", ""},
		{"@malformed", refs.KindInvalid, "", ""},
		// Retired colon-form local schemes are invalid under the new
		// path-style grammar.
		{"skill:focus", refs.KindInvalid, "", ""},
		{"tool:my-parser", refs.KindInvalid, "", ""},
		{"hook:pre-spawn", refs.KindInvalid, "", ""},
		{"local:my-parser", refs.KindInvalid, "", ""},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := refs.ParseRef(c.in)
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
		{"skill/focus", "skill/focus", ""},
		{"tool/my-parser@0.1", "tool/my-parser", "0.1"},
		{"github:acme/foo@1.2.3", "github:acme/foo", "1.2.3"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			dependency, version := refs.SplitVersion(c.in)
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

	got := refs.ResolveTool(root, refs.ParseRef("tool/present"), nil, false)
	if got != refs.ResolveOK {
		t.Errorf("present local tool: want OK, got %v", got)
	}

	got = refs.ResolveTool(root, refs.ParseRef("tool/missing"), nil, false)
	if got != refs.ResolveNotFound {
		t.Errorf("missing local tool: want NotFound, got %v", got)
	}
}

func TestResolveTool_LocalSkill(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "spwn", "skills"))
	if err := os.WriteFile(filepath.Join(root, "spwn", "skills", "focus.md"), []byte("# focus"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := refs.ResolveTool(root, refs.ParseRef("skill/focus"), nil, false)
	if got != refs.ResolveOK {
		t.Errorf("present local skill: want OK, got %v", got)
	}

	got = refs.ResolveTool(root, refs.ParseRef("skill/missing"), nil, false)
	if got != refs.ResolveNotFound {
		t.Errorf("missing local skill: want NotFound, got %v", got)
	}
}

func TestResolveTool_LocalHook(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "spwn", "hooks"))
	body := []byte("event: SessionStart\ncommand: echo hi\n")
	if err := os.WriteFile(filepath.Join(root, "spwn", "hooks", "pre-spawn.yaml"), body, 0o644); err != nil {
		t.Fatal(err)
	}

	got := refs.ResolveTool(root, refs.ParseRef("hook/pre-spawn"), nil, false)
	if got != refs.ResolveOK {
		t.Errorf("present local hook: want OK, got %v", got)
	}

	got = refs.ResolveTool(root, refs.ParseRef("hook/missing"), nil, false)
	if got != refs.ResolveNotFound {
		t.Errorf("missing local hook: want NotFound, got %v", got)
	}
}

func TestResolveTool_InvalidBareName(t *testing.T) {
	root := t.TempDir()
	// Even if a file named "present" existed in spwn/tools/, a bare
	// ref must NOT resolve — the grammar rejects it up front.
	mustMkdir(t, filepath.Join(root, "spwn", "tools", "present"))

	got := refs.ResolveTool(root, refs.ParseRef("present"), nil, false)
	if got != refs.ResolveInvalid {
		t.Errorf("bare ref: want ResolveInvalid, got %v", got)
	}
}

func TestResolveTool_RetiredColonForm(t *testing.T) {
	root := t.TempDir()
	// The retired `tool:foo` colon-form must not resolve, even if the
	// matching directory exists. The grammar gates it as invalid so
	// the user gets a hint pointing at the new form.
	mustMkdir(t, filepath.Join(root, "spwn", "tools", "present"))

	for _, in := range []string{"tool:present", "skill:focus", "hook:pre-spawn"} {
		if got := refs.ResolveTool(root, refs.ParseRef(in), nil, false); got != refs.ResolveInvalid {
			t.Errorf("retired form %q: want ResolveInvalid, got %v", in, got)
		}
	}
}

func TestResolveTool_Builtin(t *testing.T) {
	builtin := map[string]struct{}{
		"spwn:unix": {},
		"spwn:git":  {},
	}

	got := refs.ResolveTool("", refs.ParseRef("spwn:unix"), builtin, true)
	if got != refs.ResolveOK {
		t.Errorf("known builtin: want OK, got %v", got)
	}

	got = refs.ResolveTool("", refs.ParseRef("spwn:nonesuch"), builtin, true)
	if got != refs.ResolveNotFound {
		t.Errorf("unknown builtin with catalog: want NotFound, got %v", got)
	}

	got = refs.ResolveTool("", refs.ParseRef("spwn:nonesuch"), nil, false)
	if got != refs.ResolveOK {
		t.Errorf("unknown builtin without catalog: want OK (permissive), got %v", got)
	}
}

func TestResolveTool_Registry(t *testing.T) {
	got := refs.ResolveTool("", refs.ParseRef("github:acme/foo"), nil, false)
	if got != refs.ResolveRegistryUnsupported {
		t.Errorf("registry ref: want RegistryUnsupported, got %v", got)
	}
}

func TestResolveSkill_PathForm(t *testing.T) {
	root := t.TempDir()

	// skill/ form — file.
	mustMkdir(t, filepath.Join(root, "spwn", "skills"))
	if err := os.WriteFile(filepath.Join(root, "spwn", "skills", "focus.md"), []byte("# focus"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := refs.ResolveSkill(root, refs.ParseRef("skill/focus"), nil, false)
	if got != refs.ResolveOK {
		t.Errorf("skill/ form resolves: want OK, got %v", got)
	}

	// tool/ form — directory.
	mustMkdir(t, filepath.Join(root, "spwn", "tools", "debug"))
	got = refs.ResolveSkill(root, refs.ParseRef("tool/debug"), nil, false)
	if got != refs.ResolveOK {
		t.Errorf("tool/ form resolves: want OK, got %v", got)
	}

	// Bare name is rejected outright.
	got = refs.ResolveSkill(root, refs.ParseRef("focus"), nil, false)
	if got != refs.ResolveInvalid {
		t.Errorf("bare ref via ResolveSkill: want ResolveInvalid, got %v", got)
	}

	got = refs.ResolveSkill(root, refs.ParseRef("skill/missing"), nil, false)
	if got != refs.ResolveNotFound {
		t.Errorf("missing skill/ form: want NotFound, got %v", got)
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}
