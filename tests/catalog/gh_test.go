package catalog_test

import (
	"strings"
	"testing"
)

// TestGh_WrapperUsesGhAuthToken locks in that the gh-mcp wrapper
// derives its credential from `gh auth token` (which reads
// $GH_CONFIG_DIR/hosts.yml — bind-mounted by spwn) rather than
// expecting an env-var. This is the regression-trap for re-
// introducing the GITHUB_PERSONAL_ACCESS_TOKEN env passthrough
// path, which was deleted along with the migration to
// `spwn auth login github`.
func TestGh_WrapperUsesGhAuthToken(t *testing.T) {
	tool := findTool("spwn:gh")
	if tool == nil {
		t.Skip("spwn:gh not registered")
	}
	all := strings.Join(tool.Install().Commands, "\n")
	mustContain := []string{
		"gh-mcp",                                  // wrapper binary name
		"gh auth token",                           // reads from gh config, not env
		"spwn auth login github",                  // hint when missing
		"mcp2cli",                                 // exec target
		"--mcp-stdio",                             // GitHub stdio MCP server
		"mcp-server-github",                       // server binary
		"GITHUB_PERSONAL_ACCESS_TOKEN",            // forwarded to mcp2cli's child
	}
	for _, want := range mustContain {
		if !strings.Contains(all, want) {
			t.Errorf("gh wrapper missing expected substring %q", want)
		}
	}
	// Must NOT depend on the host env var being pre-set; the
	// wrapper sets it from `gh auth token` itself.
	if strings.Contains(all, `"$GITHUB_PERSONAL_ACCESS_TOKEN"`) {
		t.Errorf("gh wrapper should not reference $GITHUB_PERSONAL_ACCESS_TOKEN as input — derive via `gh auth token`")
	}
	if strings.Contains(all, "if [ -z \"$GITHUB_PERSONAL_ACCESS_TOKEN\"") {
		t.Errorf("gh wrapper must not gate on env var presence — gate on `gh auth token` exit code instead")
	}
}
