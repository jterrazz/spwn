package dockerfile

import (
	"strings"
	"testing"
)

func TestGenerate_EmptyTools(t *testing.T) {
	base := []byte("FROM ubuntu:24.04\nUSER spwn\n")
	result := Generate(base, nil, "1.0.0")
	if !strings.Contains(string(result), "FROM ubuntu:24.04") {
		t.Error("expected base Dockerfile content")
	}
	if !strings.Contains(string(result), "sh.spwn.image-version") {
		t.Error("expected version label")
	}
}

func TestGenerate_MergesAptPackages(t *testing.T) {
	base := []byte("FROM ubuntu:24.04\n")
	tools := []ToolInput{
		{Name: "@unix", Kind: "sdk", Packages: []string{"bash", "grep", "curl"}},
		{Name: "@git", Kind: "tool", Packages: []string{"git", "curl"}}, // curl is duplicate
	}

	result := string(Generate(base, tools, ""))
	count := strings.Count(result, "apt-get install")
	if count != 1 {
		t.Errorf("expected 1 apt-get install, got %d", count)
	}
	if strings.Count(result, "curl") != 1 {
		t.Error("expected curl to be deduplicated")
	}
}

func TestGenerate_OrdersToolSections(t *testing.T) {
	base := []byte("FROM ubuntu:24.04\n")
	tools := []ToolInput{
		{Name: "@node", Kind: "sdk", Commands: []string{"curl -fsSL https://deb.nodesource.com/setup_20.x | bash -"}},
		{Name: "@qmd", Kind: "tool", Commands: []string{"npm install -g @tobilu/qmd"}},
	}

	result := string(Generate(base, tools, ""))
	nodeIdx := strings.Index(result, "# @node")
	qmdIdx := strings.Index(result, "# @qmd")
	if nodeIdx >= qmdIdx {
		t.Error("@node section should come before @qmd section")
	}
}

func TestGenerate_IncludesEnv(t *testing.T) {
	base := []byte("FROM ubuntu:24.04\n")
	tools := []ToolInput{
		{Name: "@node", Kind: "sdk", Env: map[string]string{"NODE_ENV": "production"}},
	}

	result := string(Generate(base, tools, ""))
	if !strings.Contains(result, "ENV NODE_ENV=production") {
		t.Error("expected ENV directive")
	}
}
