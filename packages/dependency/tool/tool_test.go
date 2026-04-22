package tool

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestPackages_UnmarshalYAML_valid exercises the happy paths: a full
// apt block, a bare `apt: []`, and an explicitly null `packages:`
// scalar (yaml's way to express "nothing under this key"). All three
// must round-trip into a zero- or near-zero Packages without error.
func TestPackages_UnmarshalYAML_valid(t *testing.T) {
	cases := []struct {
		name    string
		yaml    string
		wantApt []string
	}{
		{
			name: "populated apt list",
			yaml: `apt:
  - python3
  - python3-pip`,
			wantApt: []string{"python3", "python3-pip"},
		},
		{
			name:    "empty apt list",
			yaml:    "apt: []",
			wantApt: nil,
		},
		{
			name:    "explicit null",
			yaml:    "null",
			wantApt: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var p Packages
			if err := yaml.Unmarshal([]byte(tc.yaml), &p); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !equalStrings(p.Apt, tc.wantApt) {
				t.Errorf("Apt = %v, want %v", p.Apt, tc.wantApt)
			}
		})
	}
}

// TestPackages_UnmarshalYAML_unknownManager is the whole point of
// UnmarshalYAML: catch typos and not-yet-supported managers at parse
// time so a tool.yaml's packages block can't silently evaporate. The
// error must name the offending key so the user knows where to look.
func TestPackages_UnmarshalYAML_unknownManager(t *testing.T) {
	cases := []struct {
		name     string
		yaml     string
		wantText string
	}{
		{
			name:     "apt typo",
			yaml:     `apy: [python3]`,
			wantText: "unknown package manager \"apy\"",
		},
		{
			name:     "not-yet-supported manager",
			yaml:     `apk: [python3]`,
			wantText: "unknown package manager \"apk\"",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var p Packages
			err := yaml.Unmarshal([]byte(tc.yaml), &p)
			if err == nil {
				t.Fatalf("expected error, got nil (Packages=%+v)", p)
			}
			if !strings.Contains(err.Error(), tc.wantText) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.wantText)
			}
		})
	}
}

// TestPackages_UnmarshalYAML_wrongShape rejects the legacy flat list
// form (`packages: [python3, ...]`) at the type boundary — the key
// is now a keyed map, and a flat list is almost certainly a project
// that was never migrated off the old schema.
func TestPackages_UnmarshalYAML_wrongShape(t *testing.T) {
	var p Packages
	err := yaml.Unmarshal([]byte(`[python3, python3-pip]`), &p)
	if err == nil {
		t.Fatalf("expected error for flat list shape, got nil")
	}
	if !strings.Contains(err.Error(), "want a mapping") {
		t.Errorf("error = %q, want hint about mapping shape", err.Error())
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
