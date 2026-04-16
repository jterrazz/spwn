package architect

import (
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
