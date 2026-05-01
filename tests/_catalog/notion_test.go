package catalog_test

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"spwn.sh/packages/dependency"
	"spwn.sh/packages/dependency/tool"
)

// findTool returns the registered tool with the given name, or nil.
func findTool(name string) tool.Tool {
	for _, t := range dependency.BuiltinTools() {
		if t.Name() == name {
			return t
		}
	}
	return nil
}

// TestNotion_DependsOnMcp2cli locks in that spwn:notion pulls in
// the mcp2cli foundation tool. Without it, the wrapper exec'd
// inside the world has no `mcp2cli` on PATH and the agent gets
// "command not found" with no actionable hint.
func TestNotion_DependsOnMcp2cli(t *testing.T) {
	tool := findTool("spwn:notion")
	if tool == nil {
		t.Fatal("spwn:notion not registered in builtin catalog")
	}
	deps := tool.Dependencies()
	found := false
	for _, d := range deps {
		if d == "spwn:mcp2cli" {
			found = true
		}
	}
	if !found {
		t.Errorf("spwn:notion must depend on spwn:mcp2cli, got %v", deps)
	}
}

// TestNotion_NoNodeDependency catches the regression where someone
// re-introduces `npm install -g @notionhq/notion-mcp-server`. The
// hosted MCP at mcp.notion.com handles the protocol — local stdio
// server is not used and node is not needed.
func TestNotion_NoNodeDependency(t *testing.T) {
	tool := findTool("spwn:notion")
	if tool == nil {
		t.Skip("spwn:notion not registered")
	}
	for _, d := range tool.Dependencies() {
		if d == "spwn:node" {
			t.Errorf("spwn:notion should not depend on spwn:node — hosted MCP needs no local server, got deps %v", tool.Dependencies())
		}
	}
	for _, cmd := range tool.Install().Commands {
		if strings.Contains(cmd, "npm install") || strings.Contains(cmd, "@notionhq/notion-mcp-server") {
			t.Errorf("spwn:notion install must not pull npm packages (hosted MCP), got %q", cmd)
		}
	}
}

// TestNotion_WrapperHasOAuthGate locks the wrapper's auth gate:
// it must check for tokens.json and point the user at
// `spwn auth login notion` rather than expecting a NOTION_TOKEN
// env var (the old route).
func TestNotion_WrapperHasOAuthGate(t *testing.T) {
	tool := findTool("spwn:notion")
	if tool == nil {
		t.Skip("spwn:notion not registered")
	}
	all := strings.Join(tool.Install().Commands, "\n")
	mustContain := []string{
		"notion-mcp",                      // wrapper binary name
		"tokens.json",                     // checks for cached OAuth token
		"spwn auth login notion",          // hint when missing
		"https://mcp.notion.com/mcp",      // hosted MCP URL
		"mcp2cli",                         // exec target
		"--oauth",                         // OAuth transport flag
	}
	for _, want := range mustContain {
		if !strings.Contains(all, want) {
			t.Errorf("notion install commands missing expected substring %q", want)
		}
	}
	// Must NOT use the old env-var path.
	mustNotContain := []string{
		"NOTION_TOKEN",
		"OPENAPI_MCP_HEADERS",
	}
	for _, bad := range mustNotContain {
		if strings.Contains(all, bad) {
			t.Errorf("notion install must not reference legacy auth path %q (replaced by OAuth bind-mount)", bad)
		}
	}
}

// TestNotion_WrapperHashMatchesProviderKey locks the URL-hash
// sentinel in the wrapper: the bash sha256sum + cut must produce
// the same key mcp2cli (and packages/auth/mcp.providerKey) use,
// otherwise the `if [ -f ... ]` check looks at the wrong path
// and login never appears authenticated inside the world.
func TestNotion_WrapperHashMatchesProviderKey(t *testing.T) {
	tool := findTool("spwn:notion")
	if tool == nil {
		t.Skip("spwn:notion not registered")
	}
	const url = "https://mcp.notion.com/mcp"
	sum := sha256.Sum256([]byte(url))
	want := hex.EncodeToString(sum[:])[:16]

	all := strings.Join(tool.Install().Commands, "\n")
	// We assert the formula is in the script (sha256sum | cut -c1-16
	// over the literal URL). If someone changes the cut to -c1-12 or
	// the URL string drifts, the in-world cache lookup goes blind.
	if !strings.Contains(all, "sha256sum") || !strings.Contains(all, "cut -c1-16") {
		t.Errorf("notion wrapper must hash the MCP URL via 'sha256sum | cut -c1-16' to match mcp2cli's storage key; got:\n%s", all)
	}
	if !strings.Contains(all, url) {
		t.Errorf("notion wrapper must hash the literal URL %q (so the bash + Go hashes match); got:\n%s", url, all)
	}
	// Sanity: confirm the hash for our URL is what we expect today.
	// Drift here ⇒ either url constant moved (other test catches it)
	// or sha256 stopped being sha256 (the world ended).
	const wantSentinel = "1cbd18bf1818c780"
	if want != wantSentinel {
		t.Errorf("provider key sentinel drifted: got %q, expected %q", want, wantSentinel)
	}
}
