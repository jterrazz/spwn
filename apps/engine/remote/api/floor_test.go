package api_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/remote/api"
)

func TestAReleaseBeforeTheFloorIsRefusedAndABuildOfNoReleaseIsNot(t *testing.T) {
	for version, want := range map[string]bool{
		"v3.1.0":            true,
		"v3.0.9":            true,
		"v2.9.0":            true,
		"v3.2.0":            false,
		"v3.10.0":           false,
		"v4.0.0":            false,
		"v3.1.0-14-gabcdef": false,
		"v3.1.0-dirty":      false,
		"dev":               false,
		"":                  false,
		"dry-run-abcdef":    false,
	} {
		if got := api.BeforeFloor(version); got != want {
			t.Errorf("BeforeFloor(%q) = %v, want %v", version, got, want)
		}
	}
}
