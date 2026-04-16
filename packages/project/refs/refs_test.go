package refs_test

import (
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/project/refs"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in    string
		kind  refs.Kind
		owner string
		name  string
	}{
		{"local-tool", refs.KindLocal, "", "local-tool"},
		{"  spaced  ", refs.KindLocal, "", "spaced"},
		{"@spwn/unix", refs.KindSpwnBuiltin, "spwn", "unix"},
		{"@spwn/claude-code", refs.KindSpwnBuiltin, "spwn", "claude-code"},
		{"@acme/foo", refs.KindRegistry, "acme", "foo"},
		{"@jterrazz/python", refs.KindRegistry, "jterrazz", "python"},
		{"@malformed", refs.KindRegistry, "malformed", ""},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := refs.Parse(c.in)
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
			pack, version := refs.SplitVersion(c.in)
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
	mustMkdir(t, filepath.Join(root, "spwn", "plugins", "present"))

	got := refs.ResolveTool(root, refs.Parse("present"), nil, false)
	if got != refs.ResolveOK {
		t.Errorf("present local pack: want OK, got %v", got)
	}

	got = refs.ResolveTool(root, refs.Parse("missing"), nil, false)
	if got != refs.ResolveNotFound {
		t.Errorf("missing local pack: want NotFound, got %v", got)
	}
}

func TestResolveTool_Builtin(t *testing.T) {
	builtin := map[string]struct{}{
		"@spwn/unix": {},
		"@spwn/git":  {},
	}

	got := refs.ResolveTool("", refs.Parse("@spwn/unix"), builtin, true)
	if got != refs.ResolveOK {
		t.Errorf("known builtin: want OK, got %v", got)
	}

	got = refs.ResolveTool("", refs.Parse("@spwn/nonesuch"), builtin, true)
	if got != refs.ResolveNotFound {
		t.Errorf("unknown builtin with catalog: want NotFound, got %v", got)
	}

	got = refs.ResolveTool("", refs.Parse("@spwn/nonesuch"), nil, false)
	if got != refs.ResolveOK {
		t.Errorf("unknown builtin without catalog: want OK (permissive), got %v", got)
	}
}

func TestResolveTool_Registry(t *testing.T) {
	got := refs.ResolveTool("", refs.Parse("@acme/foo"), nil, false)
	if got != refs.ResolveRegistryUnsupported {
		t.Errorf("registry ref: want RegistryUnsupported, got %v", got)
	}
}

func TestResolveSkill_Local(t *testing.T) {
	root := t.TempDir()

	// File-form skill.
	mustMkdir(t, filepath.Join(root, "spwn", "plugins"))
	if err := os.WriteFile(filepath.Join(root, "spwn", "plugins", "focus.md"), []byte("# focus"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := refs.ResolveSkill(root, refs.Parse("focus"), nil, false)
	if got != refs.ResolveOK {
		t.Errorf("file-form skill: want OK, got %v", got)
	}

	// Directory-form skill.
	mustMkdir(t, filepath.Join(root, "spwn", "plugins", "debug"))

	got = refs.ResolveSkill(root, refs.Parse("debug"), nil, false)
	if got != refs.ResolveOK {
		t.Errorf("dir-form skill: want OK, got %v", got)
	}

	got = refs.ResolveSkill(root, refs.Parse("missing"), nil, false)
	if got != refs.ResolveNotFound {
		t.Errorf("missing skill: want NotFound, got %v", got)
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}
