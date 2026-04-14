package architect

import (
	"testing"
)

func TestMergeUnique(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want []string
	}{
		{
			name: "no_overlap",
			a:    []string{"a", "b"},
			b:    []string{"c", "d"},
			want: []string{"a", "b", "c", "d"},
		},
		{
			name: "with_overlap",
			a:    []string{"a", "b", "c"},
			b:    []string{"b", "c", "d"},
			want: []string{"a", "b", "c", "d"},
		},
		{
			name: "empty_first",
			a:    nil,
			b:    []string{"x", "y"},
			want: []string{"x", "y"},
		},
		{
			name: "empty_second",
			a:    []string{"x", "y"},
			b:    nil,
			want: []string{"x", "y"},
		},
		{
			name: "both_empty",
			a:    nil,
			b:    nil,
			want: nil,
		},
		{
			name: "duplicates_within_single_slice",
			a:    []string{"a", "a", "b"},
			b:    []string{"b", "c"},
			want: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeUnique(tt.a, tt.b)
			if len(got) != len(tt.want) {
				t.Errorf("mergeUnique() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("mergeUnique()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
