package manifest

import (
	"testing"

	"spwn.sh/packages/world/internal/models"
)

func TestExpandTools(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "unix_pack",
			in:   []string{"@spwn/unix"},
			want: ToolPacks["@spwn/unix"],
		},
		{
			name: "git_pack",
			in:   []string{"@spwn/git"},
			want: []string{"git"},
		},
		{
			name: "mixed_packs_and_individual",
			in:   []string{"@spwn/git", "custom-tool", "bash"},
			want: []string{"git", "custom-tool", "bash"},
		},
		{
			name: "deduplication",
			in:   []string{"@spwn/git", "git"},
			want: []string{"git"},
		},
		{
			name: "empty_list",
			in:   nil,
			want: nil,
		},
		{
			name: "unknown_pack_treated_as_tool",
			in:   []string{"@nonexistent"},
			want: []string{"@nonexistent"},
		},
		{
			name: "multiple_packs_overlap",
			in:   []string{"@spwn/unix", "bash"},
			// bash is in @spwn/unix, so it should not appear twice
			want: ToolPacks["@spwn/unix"],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandTools(tt.in)
			if len(got) != len(tt.want) {
				t.Errorf("ExpandTools(%v) = %v (len %d), want %v (len %d)",
					tt.in, got, len(got), tt.want, len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ExpandTools(%v)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
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
	})

	t.Run("does_not_overwrite_set_values", func(t *testing.T) {
		m := models.Manifest{
			Physics: models.PhysicsManifest{
				Constants: models.ConstantsManifest{
					CPU:    4,
					Memory: "2g",
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
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		m       models.Manifest
		wantErr bool
	}{
		{
			name: "valid_manifest",
			m: models.Manifest{
				Physics: models.PhysicsManifest{
					Constants: models.ConstantsManifest{CPU: 1},
				},
			},
		},
		{
			name: "negative_cpu",
			m: models.Manifest{
				Physics: models.PhysicsManifest{
					Constants: models.ConstantsManifest{CPU: -1},
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
