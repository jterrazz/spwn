package backend

import (
	"testing"
)

func TestNeedsRebuild(t *testing.T) {
	tests := []struct {
		name            string
		currentVersion  string
		expectedVersion string
		want            bool
	}{
		{"missing image needs build", "", "1.0.0", true},
		{"matching version skips build", "1.0.0", "1.0.0", false},
		{"version mismatch triggers rebuild", "0.9.0", "1.0.0", true},
		{"both empty triggers rebuild", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NeedsRebuild(tt.currentVersion, tt.expectedVersion)
			if got != tt.want {
				t.Errorf("NeedsRebuild(%q, %q) = %v, want %v",
					tt.currentVersion, tt.expectedVersion, got, tt.want)
			}
		})
	}
}
