package manifest

import (
	"testing"

	"github.com/jterrazz/spwn/core/universe/internal/models"
)

func TestExpandElements(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "unix_pack",
			in:   []string{"@unix"},
			want: ElementPacks["@unix"],
		},
		{
			name: "git_pack",
			in:   []string{"@git"},
			want: []string{"git"},
		},
		{
			name: "mixed_packs_and_individual",
			in:   []string{"@git", "custom-tool", "bash"},
			want: []string{"git", "custom-tool", "bash"},
		},
		{
			name: "deduplication",
			in:   []string{"@git", "git"},
			want: []string{"git"},
		},
		{
			name: "empty_list",
			in:   nil,
			want: nil,
		},
		{
			name: "unknown_pack_treated_as_element",
			in:   []string{"@nonexistent"},
			want: []string{"@nonexistent"},
		},
		{
			name: "multiple_packs_overlap",
			in:   []string{"@unix", "bash"},
			// bash is in @unix, so it should not appear twice
			want: ElementPacks["@unix"],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandElements(tt.in)
			if len(got) != len(tt.want) {
				t.Errorf("ExpandElements(%v) = %v (len %d), want %v (len %d)",
					tt.in, got, len(got), tt.want, len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ExpandElements(%v)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	t.Run("fills_zero_values", func(t *testing.T) {
		m := models.Manifest{}
		ApplyDefaults(&m)

		if m.Physics.Constants.CPU == 0 {
			t.Error("CPU should not be zero after ApplyDefaults")
		}
		if m.Physics.Constants.Memory == "" {
			t.Error("Memory should not be empty after ApplyDefaults")
		}
		if m.Physics.Constants.Disk == "" {
			t.Error("Disk should not be empty after ApplyDefaults")
		}
		if m.Physics.Constants.Timeout == "" {
			t.Error("Timeout should not be empty after ApplyDefaults")
		}
		if m.Physics.Laws.Network == "" {
			t.Error("Network should not be empty after ApplyDefaults")
		}
		if m.Physics.Laws.MaxProcesses == 0 {
			t.Error("MaxProcesses should not be zero after ApplyDefaults")
		}
	})

	t.Run("does_not_overwrite_set_values", func(t *testing.T) {
		m := models.Manifest{
			Physics: models.PhysicsManifest{
				Constants: models.ConstantsManifest{
					CPU:     4,
					Memory:  "2g",
					Disk:    "10g",
					Timeout: "1h",
				},
				Laws: models.LawsManifest{
					Network:      "bridge",
					MaxProcesses: 256,
				},
			},
		}
		ApplyDefaults(&m)

		if m.Physics.Constants.CPU != 4 {
			t.Errorf("CPU = %d, want 4", m.Physics.Constants.CPU)
		}
		if m.Physics.Constants.Memory != "2g" {
			t.Errorf("Memory = %q, want %q", m.Physics.Constants.Memory, "2g")
		}
		if m.Physics.Constants.Disk != "10g" {
			t.Errorf("Disk = %q, want %q", m.Physics.Constants.Disk, "10g")
		}
		if m.Physics.Constants.Timeout != "1h" {
			t.Errorf("Timeout = %q, want %q", m.Physics.Constants.Timeout, "1h")
		}
		if m.Physics.Laws.Network != "bridge" {
			t.Errorf("Network = %q, want %q", m.Physics.Laws.Network, "bridge")
		}
		if m.Physics.Laws.MaxProcesses != 256 {
			t.Errorf("MaxProcesses = %d, want 256", m.Physics.Laws.MaxProcesses)
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		m       models.Manifest
		wantErr bool
	}{
		{
			name: "valid_none_network",
			m: models.Manifest{
				Physics: models.PhysicsManifest{
					Constants: models.ConstantsManifest{CPU: 1},
					Laws:      models.LawsManifest{Network: "none"},
				},
			},
		},
		{
			name: "valid_bridge_network",
			m: models.Manifest{
				Physics: models.PhysicsManifest{
					Laws: models.LawsManifest{Network: "bridge"},
				},
			},
		},
		{
			name: "valid_host_network",
			m: models.Manifest{
				Physics: models.PhysicsManifest{
					Laws: models.LawsManifest{Network: "host"},
				},
			},
		},
		{
			name: "invalid_network",
			m: models.Manifest{
				Physics: models.PhysicsManifest{
					Laws: models.LawsManifest{Network: "custom"},
				},
			},
			wantErr: true,
		},
		{
			name: "negative_cpu",
			m: models.Manifest{
				Physics: models.PhysicsManifest{
					Constants: models.ConstantsManifest{CPU: -1},
					Laws:      models.LawsManifest{Network: "none"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.m)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
