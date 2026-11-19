package wire

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

type noRelease struct{}

func (noRelease) Read(context.Context) (engine.ReleaseKept, bool, error) {
	return engine.ReleaseKept{}, false, nil
}

func (noRelease) Write(context.Context, engine.ReleaseKept) error { return nil }

func TestOnlyAMacsEngineRunsTheNightlyUpgradeAndTheReleaseRefresh(t *testing.T) {
	system := localtest.Mac(localtest.NewRunner(t), localtest.NewFS(), localtest.Env{HomeDir: "/home/u", OS: "darwin"}, localtest.NewNet(), &localtest.Clock{})
	for kind, want := range map[catalog.MachineKind]int{catalog.Mac: 2, catalog.Linux: 0, catalog.MacOS: 0} {
		m := machine{kind: kind, sys: system, stores: &stores{release: noRelease{}}}
		if got := len(m.macJobs(nil)); got != want {
			t.Errorf("%s engine runs %d Mac-only jobs; want %d", kind, got, want)
		}
	}
}
