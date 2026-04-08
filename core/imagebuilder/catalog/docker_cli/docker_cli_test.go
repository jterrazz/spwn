package docker_cli

import "testing"

func TestDockerCLI_Name(t *testing.T) {
	if Tool.Name() != "@docker-cli" {
		t.Errorf("expected @docker-cli, got %s", Tool.Name())
	}
}

func TestDockerCLI_HasVerify(t *testing.T) {
	if len(Tool.Verify()) == 0 {
		t.Error("@docker-cli should have verify commands")
	}
}
