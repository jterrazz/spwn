package api_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/remote/api"
)

func TestAReleaseBeforeTheFloorIsRefusedAndABuildOfNoReleaseIsNot(t *testing.T) {
	for version, want := range map[string]bool{
		"v0.0.9":            true,
		"v0.0.0":            true,
		"v0.1.0":            false,
		"v0.1.1":            false,
		"v0.10.0":           false,
		"v1.0.0":            false,
		"v0.1.0-alpha.1":    false,
		"v0.0.9-14-gabcdef": false,
		"v0.0.9-dirty":      false,
		"dev":               false,
		"":                  false,
		"dry-run-abcdef":    false,
	} {
		if got := api.BeforeFloor(version); got != want {
			t.Errorf("BeforeFloor(%q) = %v, want %v", version, got, want)
		}
	}
}
