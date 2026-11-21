package machines_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines"
)

func TestAPeerIsServedOnlyOnThisStationsRelease(t *testing.T) {
	const here = "v0.21.0-alpha.2"
	for far, want := range map[string]machines.Skew{
		"v0.21.0-alpha.2":                   machines.Speaks,
		"v0.21.0-alpha.2-14-gabcdef1":       machines.Speaks,
		"v0.21.0-alpha.2-14-gabcdef1-dirty": machines.Speaks,
		"v0.21.0-alpha.2-dirty":             machines.Speaks,
		"v0.21.0-alpha.1":                   machines.FarBehind,
		"v0.20.0":                           machines.FarBehind,
		"v0.20.0-3-gabcdef1":                machines.FarBehind,
		"v0.0.9":                            machines.FarBehind,
		"v3.2.0":                            machines.FarBehind,
		"v3.0.0-5-gabcdef1":                 machines.FarBehind,
		"v1.0.0":                            machines.FarBehind,
		"v0.21.0-alpha.3":                   machines.HereBehind,
		"v0.21.0":                           machines.HereBehind,
		"v0.22.0":                           machines.HereBehind,
		"dev":                               machines.Speaks,
		"":                                  machines.Speaks,
		"dry-run-1018bfdf9":                 machines.Speaks,
		"c09ac16ff":                         machines.Speaks,
		"c09ac16ff-dirty":                   machines.Speaks,
	} {
		if got := machines.SkewOf(here, far); got != want {
			t.Errorf("SkewOf(%q, %q) = %v, want %v", here, far, got, want)
		}
	}
}

func TestABuildOfNoReleaseServesOnlySuchBuilds(t *testing.T) {
	for _, here := range []string{"dev", "", "dry-run-1018bfdf9", "c09ac16ff"} {
		if got := machines.SkewOf(here, "v0.21.0-alpha.2"); got != machines.HereBehind {
			t.Errorf("SkewOf(%q, a release) = %v, want HereBehind", here, got)
		}
		if got := machines.SkewOf(here, "dev"); got != machines.Speaks {
			t.Errorf("SkewOf(%q, dev) = %v, want Speaks", here, got)
		}
	}
}

func TestAStationOfTheOldNumberingIsBehindOneOfTheNewLine(t *testing.T) {
	if got := machines.SkewOf("v3.3.0", "v3.2.0"); got != machines.FarBehind {
		t.Errorf("SkewOf(v3.3.0, v3.2.0) = %v, want FarBehind", got)
	}
	if got := machines.SkewOf("v3.2.0", "v3.3.0"); got != machines.HereBehind {
		t.Errorf("SkewOf(v3.2.0, v3.3.0) = %v, want HereBehind", got)
	}
}
