package backend

import (
	"fmt"
	"strings"
	"testing"

	"spwn.sh/core/foundation"
	"spwn.sh/platform/images"
)

// TestDockerfileVersionMatchesConstant verifies that the LABEL in each embedded
// Dockerfile is consistent with the corresponding Go version constant. This
// catches cases where a developer bumps the constant but forgets to update the
// Dockerfile (or vice versa).
func TestDockerfileVersionMatchesConstant(t *testing.T) {
	cases := []struct {
		name            string
		dockerfile      []byte
		expectedVersion string
	}{
		{"Dockerfile (world)", images.Dockerfile, foundation.WorldImageVersion},
		{"Dockerfile.architect", images.DockerfileArchitect, foundation.ArchitectImageVersion},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			version := extractLabelVersion(string(tc.dockerfile), foundation.ImageVersionLabel)
			if version == "" {
				t.Fatalf("no LABEL %s found in %s", foundation.ImageVersionLabel, tc.name)
			}
			if version != tc.expectedVersion {
				t.Errorf("version mismatch in %s: Dockerfile has %q, Go constant has %q",
					tc.name, version, tc.expectedVersion)
			}
		})
	}
}

// extractLabelVersion parses a Dockerfile string for a LABEL line with the
// given key and returns the value. Supports both LABEL key="value" and
// LABEL key=value forms.
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
