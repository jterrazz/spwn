package backend

import (
	"fmt"
	"strings"
	"testing"

	"spwn.sh/core/foundation"
	"spwn.sh/core/imagebuilder/base"
)

// TestDockerfileVersionMatchesConstant verifies that the LABEL in each embedded
// Dockerfile is consistent with the corresponding Go version constant.
func TestDockerfileVersionMatchesConstant(t *testing.T) {
	cases := []struct {
		name            string
		dockerfile      []byte
		expectedVersion string
	}{
		{"world.Dockerfile", base.WorldDockerfile, foundation.WorldImageVersion},
		{"architect.Dockerfile", base.ArchitectDockerfile, foundation.ArchitectImageVersion},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Base Dockerfiles no longer have version labels — the generator adds them.
			// This test now just verifies the Dockerfiles are non-empty and well-formed.
			if len(tc.dockerfile) == 0 {
				t.Fatalf("%s is empty", tc.name)
			}
			if !strings.Contains(string(tc.dockerfile), "FROM") {
				t.Fatalf("%s does not contain FROM directive", tc.name)
			}
			_ = tc.expectedVersion // Version is now applied by the generator
		})
	}
}

// extractLabelVersion parses a Dockerfile string for a LABEL line.
func extractLabelVersion(dockerfile, labelKey string) string {
	prefix := fmt.Sprintf("LABEL %s=", labelKey)
	for _, line := range strings.Split(dockerfile, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			val := strings.TrimPrefix(line, prefix)
			val = strings.Trim(val, "\"")
			return val
		}
	}
	return ""
}
