package claudecode

import (
	"strings"
	"testing"
)

// TestSpawner_ContainerConfigPath pins the container-side settings
// path Claude Code reads on startup. The architect's runtime-config
// injector reads this to know WHERE to merge per-tool MCP config; a
// regression here silently breaks every tool whose runtime-config
// block targets spwn:claude-code.
func TestSpawner_ContainerConfigPath(t *testing.T) {
	got := Spawner.ContainerConfigPath()
	if got != "/home/spwn/.claude/settings.json" {
		t.Errorf("ContainerConfigPath() = %q, want /home/spwn/.claude/settings.json", got)
	}
}

// TestSpawner_PrelaunchShell is pure container-side plumbing now —
// the outer composer (daemon, talk.go) owns `source /credentials/
// .env`. The adapter's job is the claude-specific cred copy.
// Pinning this layering prevents a future refactor from silently
// re-introducing duplicate env sourcing when adapters get chained.
func TestSpawner_PrelaunchShell(t *testing.T) {
	got := Spawner.PrelaunchShell()

	if got == "" {
		t.Fatal("PrelaunchShell should wire claude credentials; got empty")
	}
	if strings.Contains(got, "source /credentials/.env") {
		t.Errorf("PrelaunchShell must not source /credentials/.env (outer composer owns env loading); got: %s", got)
	}
	// Spot-check the credential copy lands in the claude-specific
	// location. The actual shell quoting is golden-tested by the
	// integration path.
	for _, want := range []string{
		"/credentials/anthropic/.credentials.json",
		"$HOME/.claude",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("PrelaunchShell missing %q; got: %s", want, got)
		}
	}
}

// TestAdapter pins the claude-code umbrella: all three facets wired
// (Tool, Render, Spawn). This is the fullest-shape adapter and the
// one every other runtime's completeness is measured against.
func TestAdapter(t *testing.T) {
	if Adapter.Name != "claude-code" {
		t.Errorf("Adapter.Name = %q, want claude-code", Adapter.Name)
	}
	if Adapter.CatalogRef != "spwn:claude-code" {
		t.Errorf("Adapter.CatalogRef = %q, want spwn:claude-code", Adapter.CatalogRef)
	}
	if Adapter.DefaultProvider != "anthropic" {
		t.Errorf("Adapter.DefaultProvider = %q, want anthropic", Adapter.DefaultProvider)
	}
	if Adapter.Tool == nil {
		t.Error("Adapter.Tool is nil")
	}
	if Adapter.Render == nil {
		t.Error("Adapter.Render is nil — claude-code is the only runtime with a renderer")
	}
	if Adapter.Spawn == nil {
		t.Error("Adapter.Spawn is nil")
	}
}
