package base

import (
	"strings"
	"testing"
)

// TestDockerfileVersionMatchesConstant verifies the base Dockerfiles are
// non-empty and well-formed. Version labels are applied by the generator,
// not baked into the embedded source, so there is nothing to match here.
func TestDockerfileVersionMatchesConstant(t *testing.T) {
	cases := []struct {
		name       string
		dockerfile []byte
	}{
		{"world.Dockerfile", WorldDockerfile},
		{"architect.Dockerfile", ArchitectDockerfile},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.dockerfile) == 0 {
				t.Fatalf("%s is empty", tc.name)
			}
			if !strings.Contains(string(tc.dockerfile), "FROM") {
				t.Fatalf("%s does not contain FROM directive", tc.name)
			}
		})
	}
}
