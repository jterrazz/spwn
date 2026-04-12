package mind

import (
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/core/foundation"
)

func TestInit(t *testing.T) {
	t.Run("creates_all_layers_and_profile", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("SPWN_HOME", tmp)

		dir, err := Init("test-agent")
		if err != nil {
			t.Fatalf("Init: %v", err)
		}

		// Verify all 6 layer directories exist
		for _, layer := range foundation.MindLayers {
			layerPath := filepath.Join(dir, layer)
			info, err := os.Stat(layerPath)
			if err != nil {
				t.Errorf("expected layer %q to exist: %v", layer, err)
				continue
			}
			if !info.IsDir() {
				t.Errorf("expected %q to be a directory", layer)
			}
		}

		// Verify default profile exists
		profilePath := filepath.Join(dir, "core", "profile.md")
		if _, err := os.Stat(profilePath); err != nil {
			t.Errorf("expected default profile to exist: %v", err)
		}
	})

	t.Run("fails_if_already_exists", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("SPWN_HOME", tmp)

		_, err := Init("dup-agent")
		if err != nil {
			t.Fatalf("first Init: %v", err)
		}

		_, err = Init("dup-agent")
		if err == nil {
			t.Error("expected error for duplicate agent, got nil")
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("valid_mind_passes", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("SPWN_HOME", tmp)

		_, err := Init("valid-agent")
		if err != nil {
			t.Fatalf("Init: %v", err)
		}

		if err := Validate("valid-agent"); err != nil {
			t.Errorf("Validate: %v", err)
		}
	})

	t.Run("missing_agent_fails", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("SPWN_HOME", tmp)

		if err := Validate("nonexistent"); err == nil {
			t.Error("expected error for missing agent, got nil")
		}
	})

	t.Run("missing_core_dir_fails", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("SPWN_HOME", tmp)

		// Create agent dir without core
		agentDir := filepath.Join(tmp, "agents", "broken")
		if err := os.MkdirAll(agentDir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := Validate("broken"); err == nil {
			t.Error("expected error for missing core dir, got nil")
		}
	})
}

func TestList(t *testing.T) {
	t.Run("empty_dir_returns_nil", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("SPWN_HOME", tmp)

		agents, err := List()
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if agents != nil {
			t.Errorf("expected nil, got %v", agents)
		}
	})

	t.Run("lists_multiple_agents", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("SPWN_HOME", tmp)

		for _, name := range []string{"alpha", "beta", "gamma"} {
			if _, err := Init(name); err != nil {
				t.Fatalf("Init(%q): %v", name, err)
			}
		}

		agents, err := List()
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(agents) != 3 {
			t.Errorf("expected 3 agents, got %d", len(agents))
		}
	})

	t.Run("nonexistent_dir_returns_nil", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("SPWN_HOME", filepath.Join(tmp, "nonexistent"))

		agents, err := List()
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if agents != nil {
			t.Errorf("expected nil, got %v", agents)
		}
	})
}

func TestInit_SpecialCharactersInName(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
	}{
		{"spaces", "my agent"},
		{"unicode", "agënt-ñ"},
		{"dots", "agent.v2"},
		{"at_sign", "agent@home"},
		{"hash", "agent#1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("SPWN_HOME", tmp)

			dir, err := Init(tt.agentName)
			if err != nil {
				t.Fatalf("Init(%q): %v", tt.agentName, err)
			}

			// Should create directory and pass validation
			if _, err := os.Stat(dir); err != nil {
				t.Errorf("expected agent dir to exist: %v", err)
			}
		})
	}
}

func TestInit_VeryLongName(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	longName := ""
	for i := 0; i < 50; i++ {
		longName += "abcdefghij"
	}
	// 500-char name
	dir, err := Init(longName)
	if err != nil {
		// Some filesystems may reject very long paths — that's acceptable
		t.Logf("Init with 500-char name failed (acceptable): %v", err)
		return
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("expected agent dir to exist: %v", err)
	}
}

func TestValidate_FileInsteadOfDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	// Create a file where the agent directory should be
	agentPath := filepath.Join(tmp, "agents", "fakefile")
	os.MkdirAll(filepath.Dir(agentPath), 0755)
	os.WriteFile(agentPath, []byte("not a dir"), 0644)

	err := Validate("fakefile")
	if err == nil {
		t.Error("expected error when agent path is a file, got nil")
	}
}

func TestLayerCount(t *testing.T) {
	tests := []struct {
		name string
		info AgentInfo
		want int
	}{
		{
			name: "all_empty",
			info: AgentInfo{
				Layers: map[string][]string{
					"core": nil,
					"skills":   nil,
				},
			},
			want: 0,
		},
		{
			name: "some_with_files",
			info: AgentInfo{
				Layers: map[string][]string{
					"core": {"default.md"},
					"skills":   nil,
					"journal":  {"entry.md"},
				},
			},
			want: 2,
		},
		{
			name: "all_with_files",
			info: AgentInfo{
				Layers: map[string][]string{
					"core":  {"a.md"},
					"skills":    {"b.md"},
					"knowledge": {"c.md"},
				},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LayerCount(&tt.info)
			if got != tt.want {
				t.Errorf("LayerCount() = %d, want %d", got, tt.want)
			}
		})
	}
}
