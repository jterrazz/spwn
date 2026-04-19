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
		{Name: "spwn:unix", AptPackages: []string{"bash", "grep", "curl"}},
		{Name: "spwn:git", AptPackages: []string{"git", "curl"}}, // curl is duplicate
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
		{Name: "spwn:node", Commands: []string{"curl -fsSL https://deb.nodesource.com/setup_20.x | bash -"}},
		{Name: "spwn:qmd", Commands: []string{"npm install -g @tobilu/qmd"}},
	}

	result := string(Generate(base, tools, ""))
	nodeIdx := strings.Index(result, "# spwn:node")
	qmdIdx := strings.Index(result, "# spwn:qmd")
	if nodeIdx >= qmdIdx {
		t.Error("spwn:node section should come before spwn:qmd section")
	}
}

func TestGenerate_IncludesEnv(t *testing.T) {
	base := []byte("FROM ubuntu:24.04\n")
	tools := []ToolInput{
		{Name: "spwn:node", Env: map[string]string{"NODE_ENV": "production"}},
	}

	result := string(Generate(base, tools, ""))
	if !strings.Contains(result, "ENV NODE_ENV=production") {
		t.Error("expected ENV directive")
	}
}
