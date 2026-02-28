package physics

import (
	"strings"
	"testing"

	"github.com/jterrazz/universe/internal/config"
)

func TestGenerate_BasicConfig(t *testing.T) {
	cfg := &config.UniverseConfig{
		Image: "ubuntu:22.04",
	}
	output := Generate(cfg)

	if !strings.Contains(output, "ubuntu:22.04") {
		t.Error("output should contain image name")
	}
	if !strings.Contains(output, "# Physics") {
		t.Error("output should contain Physics header")
	}
	if !strings.Contains(output, "512m") {
		t.Error("output should contain default memory")
	}
}

func TestGenerate_WithMind(t *testing.T) {
	cfg := &config.UniverseConfig{
		Image:  "ubuntu:22.04",
		Mind:   "default",
		Memory: "1g",
		CPU:    2.0,
	}
	output := Generate(cfg)

	if !strings.Contains(output, "/mind") {
		t.Error("output should contain /mind mount")
	}
	if !strings.Contains(output, "1g") {
		t.Error("output should contain custom memory")
	}
	if !strings.Contains(output, "2.0") {
		t.Error("output should contain custom cpu")
	}
}
