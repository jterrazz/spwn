package deps_test

import (
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/deps"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in    string
		kind  deps.RefKind
		owner string
		name  string
	}{
		{"local-tool", deps.KindLocal, "", "local-tool"},
		{"  spaced  ", deps.KindLocal, "", "spaced"},
		{"@spwn/unix", deps.KindSpwnBuiltin, "spwn", "unix"},
		{"@spwn/claude-code", deps.KindSpwnBuiltin, "spwn", "claude-code"},
		{"@acme/foo", deps.KindRegistry, "acme", "foo"},
		{"@jterrazz/python", deps.KindRegistry, "jterrazz", "python"},
		{"@malformed", deps.KindRegistry, "malformed", ""},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := deps.ParseRef(c.in)
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
		pack    string
		version string
	}{
		{"@spwn/unix", "@spwn/unix", ""},
		{"@spwn/unix@24.04", "@spwn/unix", "24.04"},
		{"local-tool", "local-tool", ""},
		{"local-tool@0.1", "local-tool", "0.1"},
		{"@acme/foo@1.2.3", "@acme/foo", "1.2.3"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			pack, version := deps.SplitVersion(c.in)
			if pack != c.pack {
				t.Errorf("pack: want %q, got %q", c.pack, pack)
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

	got := deps.ResolveTool(root, deps.ParseRef("present"), nil, false)
	if got != deps.ResolveOK {
		t.Errorf("present local pack: want OK, got %v", got)
	}

	got = deps.ResolveTool(root, deps.ParseRef("missing"), nil, false)
	if got != deps.ResolveNotFound {
		t.Errorf("missing local pack: want NotFound, got %v", got)
	}
}

func TestResolveTool_Builtin(t *testing.T) {
	builtin := map[string]struct{}{
		"@spwn/unix": {},
		"@spwn/git":  {},
	}

	got := deps.ResolveTool("", deps.ParseRef("@spwn/unix"), builtin, true)
	if got != deps.ResolveOK {
		t.Errorf("known builtin: want OK, got %v", got)
	}

	got = deps.ResolveTool("", deps.ParseRef("@spwn/nonesuch"), builtin, true)
	if got != deps.ResolveNotFound {
		t.Errorf("unknown builtin with catalog: want NotFound, got %v", got)
	}

	got = deps.ResolveTool("", deps.ParseRef("@spwn/nonesuch"), nil, false)
	if got != deps.ResolveOK {
		t.Errorf("unknown builtin without catalog: want OK (permissive), got %v", got)
	}
}

func TestResolveTool_Registry(t *testing.T) {
	got := deps.ResolveTool("", deps.ParseRef("@acme/foo"), nil, false)
	if got != deps.ResolveRegistryUnsupported {
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

	got := deps.ResolveSkill(root, deps.ParseRef("focus"), nil, false)
	if got != deps.ResolveOK {
		t.Errorf("file-form skill: want OK, got %v", got)
	}

	// Directory-form skill.
	mustMkdir(t, filepath.Join(root, "spwn", "tools", "debug"))

	got = deps.ResolveSkill(root, deps.ParseRef("debug"), nil, false)
	if got != deps.ResolveOK {
		t.Errorf("dir-form skill: want OK, got %v", got)
	}

	got = deps.ResolveSkill(root, deps.ParseRef("missing"), nil, false)
	if got != deps.ResolveNotFound {
		t.Errorf("missing skill: want NotFound, got %v", got)
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}
