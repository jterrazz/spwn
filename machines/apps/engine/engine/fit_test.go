package engine_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/engine"
)

func TestFitOf(t *testing.T) {
	for _, c := range []struct {
		engine, app int
		want        engine.FitVerdict
	}{
		{5, 5, engine.Fits},
		{5, 6, engine.EngineOlder},
		{5, 4, engine.AppOlder},
		{5, 0, engine.AppOlder},
	} {
		got := engine.FitOf(c.engine, c.app)
		if got.Verdict != c.want || got.Engine != c.engine || got.App != c.app {
			t.Errorf("FitOf(%d, %d) = %+v, want %s", c.engine, c.app, got, c.want)
		}
	}
}
