package runtimeres

import (
	"strings"
	"testing"

	_ "spwn.sh/packages/runtimes/defaults"
	"spwn.sh/packages/transpile/source"
)

// isolateAuth strips every credential-bearing env var, points
// SPWN_HOME at a fresh temp dir, and redirects HOME to the same temp
// Dir so auth's file-based detectors (~/.codex/auth.json) don't leak
// The developer's real host credentials into the test.
func isolateAuth(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1")
	t.Setenv("SPWN_HOME", tmp)
	t.Setenv("HOME", tmp)
}

func TestResolve_overrideWinsEverything(t *testing.T) {
	// Override beats agent declarations AND auth state. Even with both
	// Providers authenticated the user's --runtime flag must stick.
	isolateAuth(t)
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-api")
	t.Setenv("OPENAI_API_KEY", "sk-openai")

	src := &source.ProjectSource{Agents: []source.AgentSource{
		{Name: "neo", Config: source.AgentConfig{Runtime: source.RuntimeConfig{Backend: "spwn:codex"}}},
	}}
	got, err := Resolve(src, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "claude-code" {
		t.Fatalf("override: got %q want claude-code", got)
	}
}

func TestResolve_agentDeclarationWinsOverAuth(t *testing.T) {
	// Agent pinned codex but user is also logged into Anthropic —
	// The pin must win, no auth-state ambiguity error.
	isolateAuth(t)
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-api")
	t.Setenv("OPENAI_API_KEY", "sk-openai")

	src := &source.ProjectSource{Agents: []source.AgentSource{
		{Name: "neo", Config: source.AgentConfig{Runtime: source.RuntimeConfig{Backend: "spwn:codex"}}},
	}}
	got, err := Resolve(src, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "codex" {
		t.Fatalf("agent pin: got %q want codex", got)
	}
}

func TestResolve_singleAuthProviderPicks(t *testing.T) {
	isolateAuth(t)
	t.Setenv("OPENAI_API_KEY", "sk-openai")

	// No agent pin, no override — fall through to auth. Exactly one
	// Provider connected, so we silently land on its runtime.
	src := &source.ProjectSource{Agents: []source.AgentSource{
		{Name: "neo", Config: source.AgentConfig{}},
	}}
	got, err := Resolve(src, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "codex" {
		t.Fatalf("single-auth: got %q want codex", got)
	}
}

func TestResolve_noAuthFallsBackToClaudeCode(t *testing.T) {
	isolateAuth(t)

	src := &source.ProjectSource{Agents: []source.AgentSource{
		{Name: "neo", Config: source.AgentConfig{}},
	}}
	got, err := Resolve(src, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "claude-code" {
		t.Fatalf("no-auth fallback: got %q want claude-code", got)
	}
}

func TestResolve_multipleAuthProvidersErrors(t *testing.T) {
	// The loud case. User has both providers authenticated and hasn't
	// Pinned a backend — we must surface a disambiguation hint instead
	// Of silently picking one.
	isolateAuth(t)
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-api")
	t.Setenv("OPENAI_API_KEY", "sk-openai")

	src := &source.ProjectSource{Agents: []source.AgentSource{
		{Name: "neo", Config: source.AgentConfig{}},
	}}
	_, err := Resolve(src, "")
	if err == nil {
		t.Fatal("expected ambiguity error, got nil")
	}
	for _, want := range []string{"multiple providers", "claude-code", "codex", "--runtime"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error message missing %q: %s", want, err.Error())
		}
	}
}

func TestResolve_agentConflictSurfacesBeforeAuth(t *testing.T) {
	// Two agents pin different runtimes — resolver should surface the
	// Agent-level conflict rather than fall through to auth.
	isolateAuth(t)
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-api")

	src := &source.ProjectSource{Agents: []source.AgentSource{
		{Name: "neo", Config: source.AgentConfig{Runtime: source.RuntimeConfig{Backend: "spwn:claude-code"}}},
		{Name: "morpheus", Config: source.AgentConfig{Runtime: source.RuntimeConfig{Backend: "spwn:codex"}}},
	}}
	_, err := Resolve(src, "")
	if err == nil {
		t.Fatal("expected agent-conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "conflicting runtimes") {
		t.Errorf("want agent-conflict error, got: %s", err.Error())
	}
}

func TestResolve_nilSourceHandled(t *testing.T) {
	// Legacy global-mode path: no project source. Falls straight through
	// To the auth cascade.
	isolateAuth(t)
	t.Setenv("OPENAI_API_KEY", "sk-openai")

	got, err := Resolve(nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "codex" {
		t.Fatalf("nil-source: got %q want codex", got)
	}
}
