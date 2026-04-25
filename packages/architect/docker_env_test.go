package architect

import (
	"strings"
	"testing"
)

func TestDockerEnvArgs(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-docker-test")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)
	args := DockerEnvArgs()
	found := false
	for i, a := range args {
		if a == "-e" && i+1 < len(args) && args[i+1] == "ANTHROPIC_API_KEY=sk-ant-docker-test" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ANTHROPIC_API_KEY in docker args, got %v", args)
	}
}

func TestDockerEnvVars(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-docker-test")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)
	envs := DockerEnvVars()
	found := false
	for _, e := range envs {
		if e == "ANTHROPIC_API_KEY=sk-ant-docker-test" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ANTHROPIC_API_KEY in docker envs, got %v", envs)
	}
}

// TestDockerEnvVars_NoToolEnvLeaks locks in that no tool-credential
// env vars (NOTION_TOKEN, GITHUB_PERSONAL_ACCESS_TOKEN, …) are
// forwarded. Every tool now goes through a `spwn auth login
// <provider>` cache + bind-mount; re-introducing a passthrough
// silently re-enables a foot-gun (host shell env leaking into
// world). Add new tool credentials to this list when adopting
// them so future regressions stay caught.
func TestDockerEnvVars_NoToolEnvLeaks(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("NOTION_TOKEN", "secret-should-not-leak")
	t.Setenv("GITHUB_PERSONAL_ACCESS_TOKEN", "ghp-should-not-leak")
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)

	bannedPrefixes := []string{
		"NOTION_TOKEN=",
		"GITHUB_PERSONAL_ACCESS_TOKEN=",
	}
	envs := DockerEnvVars()
	for _, e := range envs {
		for _, banned := range bannedPrefixes {
			if strings.HasPrefix(e, banned) {
				t.Errorf("%s must not be passed through (use bind-mounted cache via `spwn auth login`); got %q", banned, e)
			}
		}
	}
}
