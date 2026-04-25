package gate

import (
	"os"
	"path/filepath"
	"testing"
)

// LoadTools is the entry point the gate uses at startup to discover
// every catalog tool installed under ~/.spwn/gate/tools/. We test it
// in isolation here rather than through the full container so the
// parser's edge cases stay pinned: a malformed yaml should fail
// loud, a tool without a gate: section should be silently skipped
// (project-local tools that ship into agent containers but don't
// plug into the gate are valid), and discovery order must be stable
// for deterministic Dockerfile + provider listings.

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestLoadTools_PicksUpGateShapedTools(t *testing.T) {
	root := t.TempDir()
	// A gate-shaped tool: cookies + mcp.entry → discovered.
	writeFile(t, filepath.Join(root, "x"), "tool.yaml", `
name: spwn:x
gate:
  cookies:
    domains: [x.com, twitter.com]
    cookies: [auth_token, ct0]
  mcp:
    entry: ["node", "index.js", "mcp-serve"]
`)
	// A non-gate tool (only install steps) — should be silently
	// skipped, not error.
	writeFile(t, filepath.Join(root, "qmd"), "tool.yaml", `
name: spwn:qmd
install:
  packages: { apt: [grep] }
`)
	// A subdir without tool.yaml — also silently skipped.
	if err := os.MkdirAll(filepath.Join(root, "stale"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := LoadTools(root)
	if err != nil {
		t.Fatalf("LoadTools: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d tools, want 1: %+v", len(got), got)
	}
	if got[0].Name != "x" {
		t.Errorf("name = %q, want %q", got[0].Name, "x")
	}
	if cp := got[0].CookieProvider(); cp == nil || cp.Name != "x" {
		t.Errorf("CookieProvider = %+v, want name=x", cp)
	}
}

func TestLoadTools_SortsByName(t *testing.T) {
	root := t.TempDir()
	for _, n := range []string{"reddit", "x", "linkedin"} {
		writeFile(t, filepath.Join(root, n), "tool.yaml", `
gate:
  cookies: { domains: [example.com], cookies: [s] }
  mcp:     { entry: ["node", "i.js"] }
`)
	}
	got, err := LoadTools(root)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, t := range got {
		names = append(names, t.Name)
	}
	want := []string{"linkedin", "reddit", "x"}
	if len(names) != 3 || names[0] != want[0] || names[1] != want[1] || names[2] != want[2] {
		t.Errorf("names = %v, want %v", names, want)
	}
}

func TestLoadTools_MissingDirReturnsEmpty(t *testing.T) {
	got, err := LoadTools(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Errorf("missing dir should be no-op, got err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d tools from missing dir, want 0", len(got))
	}
}

func TestLoadTools_MalformedYAMLReturnsError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "broken"), "tool.yaml", "gate:\n  cookies: [this is not a map\n")
	got, err := LoadTools(root)
	if err == nil {
		t.Errorf("malformed yaml: want error, got %d tools", len(got))
	}
}

func TestLoadTools_NoGateSectionSilentlySkipped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "agentside"), "tool.yaml", `
name: spwn:agentside
install:
  packages: { apt: [vim] }
verify:
  - command -v vim
`)
	got, err := LoadTools(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("non-gate tool: want 0, got %d", len(got))
	}
}

func TestTool_CookieProviderNilWhenSpecEmpty(t *testing.T) {
	tl := Tool{Name: "noclookies", Spec: ToolGateSpec{MCP: &ToolMCP{Entry: []string{"x"}}}}
	if cp := tl.CookieProvider(); cp != nil {
		t.Errorf("got non-nil CookieProvider for spec without cookies: %+v", cp)
	}
}
