package process

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/engine"
)

func TestWhatTheEngineTellsOfAServerIsKeptByStation(t *testing.T) {
	for kind, want := range map[string]journal.Outcome{engine.EventRestarting: journal.Held, engine.EventLoop: journal.Held, engine.EventPoweredOff: journal.Failed, engine.EventBack: journal.Done, engine.NoticeVMReady: "", engine.EventDiskLow: ""} {
		line, kept := journaled("infra", "infra is back", engine.Told{Kind: kind})
		if kept != (want != "") || kept && (line.Outcome != want || line.By != journal.ByStation || line.Machine != "infra" || line.Text != "infra is back" || line.Subject == nil || line.Subject.Name != "infra") {
			t.Errorf("%s: %+v, kept %v; want kept %v as %q by station on infra", kind, line, kept, want != "", want)
		}
	}
}
