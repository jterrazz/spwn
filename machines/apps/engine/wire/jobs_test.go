package wire

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

// value: a machine whose binary is the engine Spwn.app carries, or the bare
// one of a Mac without the app.
func machineRunning(t *testing.T, bundled bool) machine {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, ".local", "bin", "station")
	if bundled {
		exe = filepath.Join(dir, "Applications", "Spwn.app", "Contents", "MacOS", "Spwn Engine")
	}
	if err := os.MkdirAll(filepath.Dir(exe), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	return machine{sys: local.System{Env: localtest.Env{Exe: exe, HomeDir: "/Users/o", OS: "darwin", UID: 501}}}
}

func TestTheBundlesJobsAreTheAppsOwnLabelsSoNoRecordOfTheBareJobsMatchesThem(t *testing.T) {
	m := machineRunning(t, true)
	bundle := m.bundle()
	for got, want := range map[contract.Ref]contract.Ref{
		m.engineJobRef(): {Domain: contract.Domain{UID: 501}, Label: "com.jterrazz.spwn.engine", File: bundle + "/Contents/Library/LaunchAgents/com.jterrazz.spwn.engine.plist"},
		m.agentJobRef():  {Domain: contract.Domain{UID: 501}, Label: "com.jterrazz.spwn.agent", File: bundle + "/Contents/Library/LaunchAgents/com.jterrazz.spwn.agent.plist"},
	} {
		if got != want {
			t.Errorf("the bundle's job = %+v, want %+v", got, want)
		}
	}
}

func TestTheMoveRetiresTheBareJobsByTheirLabelsAndPlists(t *testing.T) {
	m := machineRunning(t, true)
	got := m.bareJobRefs()
	want := []contract.Ref{
		{Domain: contract.Domain{UID: 501}, Label: "com.jterrazz.station.agent", File: "/Users/o/Library/LaunchAgents/com.jterrazz.station.agent.plist"},
		{Domain: contract.Domain{UID: 501}, Label: "com.jterrazz.station.engine", File: "/Users/o/Library/LaunchAgents/com.jterrazz.station.engine.plist"},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("the bare jobs = %+v, want %+v", got, want)
	}
}

func TestTheBareBinarysJobsKeepTheirLabelsAndPlists(t *testing.T) {
	m := machineRunning(t, false)
	if got := []contract.Ref{m.agentJobRef(), m.engineJobRef()}; got[0] != m.bareJobRefs()[0] || got[1] != m.bareJobRefs()[1] {
		t.Errorf("the bare binary's jobs = %+v, want %+v", got, m.bareJobRefs())
	}
}
