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
		{"local-tool", dependency.KindLocal, "", "local-tool"},
		{"  spaced  ", dependency.KindLocal, "", "spaced"},
		{"spwn:unix", dependency.KindSpwnBuiltin, "spwn", "unix"},
		{"spwn:claude-code", dependency.KindSpwnBuiltin, "spwn", "claude-code"},
		{"github:acme/foo", dependency.KindRegistry, "acme", "foo"},
		{"github:jterrazz/python", dependency.KindRegistry, "jterrazz", "python"},
		// Legacy `@owner/name` form is malformed and surfaces as an
		// empty-name KindRegistry ref so the resolver can reject it.
		{"@acme/foo", dependency.KindRegistry, "", ""},
		{"@jterrazz/python", dependency.KindRegistry, "", ""},
		{"@malformed", dependency.KindRegistry, "", ""},
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
		in      string
		dependency    string
		version string
	}{
		{"spwn:unix", "spwn:unix", ""},
		{"spwn:unix@24.04", "spwn:unix", "24.04"},
		{"local-tool", "local-tool", ""},
		{"local-tool@0.1", "local-tool", "0.1"},
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

func TestResolveTool_Local(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "spwn", "tools", "present"))

	got := dependency.ResolveTool(root, dependency.ParseRef("present"), nil, false)
	if got != dependency.ResolveOK {
		t.Errorf("present local dependency: want OK, got %v", got)
	}

	got = dependency.ResolveTool(root, dependency.ParseRef("missing"), nil, false)
	if got != dependency.ResolveNotFound {
		t.Errorf("missing local dependency: want NotFound, got %v", got)
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

func TestResolveSkill_Local(t *testing.T) {
	root := t.TempDir()

	// File-form skill.
	mustMkdir(t, filepath.Join(root, "spwn", "skills"))
	if err := os.WriteFile(filepath.Join(root, "spwn", "skills", "focus.md"), []byte("# focus"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := dependency.ResolveSkill(root, dependency.ParseRef("focus"), nil, false)
	if got != dependency.ResolveOK {
		t.Errorf("file-form skill: want OK, got %v", got)
	}

	// Directory-form skill.
	mustMkdir(t, filepath.Join(root, "spwn", "tools", "debug"))

	got = dependency.ResolveSkill(root, dependency.ParseRef("debug"), nil, false)
	if got != dependency.ResolveOK {
		t.Errorf("dir-form skill: want OK, got %v", got)
	}

	got = dependency.ResolveSkill(root, dependency.ParseRef("missing"), nil, false)
	if got != dependency.ResolveNotFound {
		t.Errorf("missing skill: want NotFound, got %v", got)
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}
