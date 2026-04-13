package architect

import (
	"testing"
)

func TestParseMemory(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{name: "megabytes_m", input: "512m", want: 512 * 1024 * 1024},
		{name: "megabytes_mb", input: "512mb", want: 512 * 1024 * 1024},
		{name: "gigabytes_g", input: "1g", want: 1024 * 1024 * 1024},
		{name: "gigabytes_gb", input: "2GB", want: 2 * 1024 * 1024 * 1024},
		{name: "kilobytes_k", input: "1024k", want: 1024 * 1024},
		{name: "kilobytes_kb", input: "1024kb", want: 1024 * 1024},
		{name: "plain_bytes", input: "4096", want: 4096},
		{name: "whitespace_trimmed", input: "  512m  ", want: 512 * 1024 * 1024},
		{name: "case_insensitive", input: "1G", want: 1024 * 1024 * 1024},
		{name: "empty_string", input: "", wantErr: true},
		{name: "invalid_string", input: "abc", wantErr: true},
		{name: "invalid_unit_suffix", input: "512x", wantErr: true},
		{name: "negative_value", input: "-1g", want: -1 * 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMemory(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseMemory(%q) expected error, got %d", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("parseMemory(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("parseMemory(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

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
