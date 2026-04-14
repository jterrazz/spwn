package validate

import (
	"os"
	"testing"
)

func TestParseRef(t *testing.T) {
	cases := []struct {
		in        string
		wantKind  RefKind
		wantOwner string
		wantName  string
	}{
		{"python", RefLocal, "", "python"},
		{"./relative", RefLocal, "", "./relative"},
		{"  local-tool  ", RefLocal, "", "local-tool"}, // trimmed
		{"@spwn/python", RefSpwnBuiltin, "spwn", "python"},
		{"@spwn/cli", RefSpwnBuiltin, "spwn", "cli"},
		{"@jterrazz/foo", RefRegistry, "jterrazz", "foo"},
		{"@community/sci", RefRegistry, "community", "sci"},
		{"@", RefRegistry, "", ""},
		{"@spwn/", RefSpwnBuiltin, "spwn", ""},
		{"@noslash", RefRegistry, "noslash", ""},
	}
	for _, c := range cases {
		got := ParseRef(c.in)
		if got.Kind != c.wantKind || got.Owner != c.wantOwner || got.Name != c.wantName {
			t.Errorf("ParseRef(%q) = {Kind:%d Owner:%q Name:%q}, want {Kind:%d Owner:%q Name:%q}",
				c.in, got.Kind, got.Owner, got.Name, c.wantKind, c.wantOwner, c.wantName)
		}
		if got.Raw != c.in {
			t.Errorf("ParseRef(%q).Raw = %q, want %q", c.in, got.Raw, c.in)
		}
	}
}

func TestResolveTool_registryAlwaysUnsupported(t *testing.T) {
	root := t.TempDir()
	builtin := map[string]struct{}{"@spwn/python": {}}

	ref := ParseRef("@jterrazz/python")
	if got := ResolveTool(root, ref, builtin, true); got != ResolveRegistryUnsupported {
		t.Errorf("@jterrazz/python: got %v, want ResolveRegistryUnsupported", got)
	}
	ref = ParseRef("@community/sci")
	if got := ResolveTool(root, ref, builtin, true); got != ResolveRegistryUnsupported {
		t.Errorf("@community/sci: got %v, want ResolveRegistryUnsupported", got)
	}
}

func TestResolveTool_spwnBuiltin(t *testing.T) {
	root := t.TempDir()
	builtin := map[string]struct{}{"@spwn/python": {}}

	if got := ResolveTool(root, ParseRef("@spwn/python"), builtin, true); got != ResolveOK {
		t.Errorf("@spwn/python with catalog: got %v, want ResolveOK", got)
	}
	if got := ResolveTool(root, ParseRef("@spwn/missing"), builtin, true); got != ResolveNotFound {
		t.Errorf("@spwn/missing with catalog: got %v, want ResolveNotFound", got)
	}
	// No catalog → permissive heuristic.
	if got := ResolveTool(root, ParseRef("@spwn/anything"), nil, false); got != ResolveOK {
		t.Errorf("@spwn/anything no catalog: got %v, want ResolveOK", got)
	}
}

func TestResolveTool_local(t *testing.T) {
	root := t.TempDir()
	// Miss: directory does not exist.
	if got := ResolveTool(root, ParseRef("nothing-here"), nil, false); got != ResolveNotFound {
		t.Errorf("local missing: got %v, want ResolveNotFound", got)
	}
	// Hit: directory exists.
	toolDir := root + "/spwn/tools/mine"
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ResolveTool(root, ParseRef("mine"), nil, false); got != ResolveOK {
		t.Errorf("local present: got %v, want ResolveOK", got)
	}
}
