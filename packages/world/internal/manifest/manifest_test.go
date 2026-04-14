package manifest

import (
	"testing"
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

