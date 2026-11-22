package wire

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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

func TestTheMoveTakesTheStandaloneMenuBarAnEarlierReleaseInstalled(t *testing.T) {
	for _, installed := range []bool{true, false} {
		m := machineRunning(t, true)
		files := localtest.NewFS()
		app := "/Users/o/Applications/Station.app"
		bin := app + "/Contents/MacOS/StationMenuBar"
		if installed {
			if err := files.WriteFile(bin, nil, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		run := localtest.NewRunner(t)
		run.On(bin, "--unregister-login-item")
		run.On("/bin/ps", "-axo", "pid=,comm=").Reply(local.Output{Stdout: "  1 /usr/sbin/syslogd\n4242 " + bin + "\n4243 " + bin + "-other\n4244 /Users/o/Applications/Other.app/Contents/MacOS/StationMenuBar\n"})
		run.Spawn(4242)
		run.Spawn(4243)
		m.sys.FS, m.sys.Run = files, run
		had, err := (engineInstaller{m: m}).retireStandaloneMenuBar(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if had != installed {
			t.Errorf("installed=%v: had = %v", installed, had)
		}
		if _, err := files.Stat(app); err == nil {
			t.Errorf("installed=%v: %s is left", installed, app)
		}
		want := 0
		if installed {
			want = 2
		}
		if installed && (run.Alive(4242) || !run.Alive(4243) || len(run.Signals) != 1) {
			t.Errorf("the standalone's process is not the one stopped: signals %v", run.Signals)
		}
		if installed && run.Calls[0].Timeout <= 0 {
			t.Errorf("the standalone binary runs with no bound: %+v", run.Calls[0])
		}
		if len(run.Calls) != want || (installed && (run.Calls[0].Args[0] != "--unregister-login-item" || run.Calls[1].Name != "/bin/ps")) {
			t.Errorf("installed=%v: ran %+v", installed, run.Calls)
		}
	}
}

func TestTheMoveReturnsWhenTheStandaloneBuildNeverExits(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, "Applications", "Station.app", "Contents", "MacOS", "StationMenuBar")
	started := filepath.Join(home, "pid")
	if err := os.MkdirAll(filepath.Dir(bin), 0o750); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\necho $$ > '" + started + "'\nwhile :; do sleep 1; done\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil { //nolint:gosec // value: a program the runner starts must be executable.
		t.Fatal(err)
	}
	m := machineRunning(t, true)
	m.sys = local.New(nil, nil, io.Discard, io.Discard)
	m.sys.Env = localtest.Env{HomeDir: home, OS: "darwin"}
	ctx, cancel := context.WithTimeout(context.Background(), standaloneMenuBarBudget+5*time.Second)
	defer cancel()
	had, err := (engineInstaller{m: m}).retireStandaloneMenuBar(ctx)
	if err != nil || !had {
		t.Fatalf("had = %v, err = %v", had, err)
	}
	if ctx.Err() != nil {
		t.Errorf("the move waited past %v for a standalone build that never exits", standaloneMenuBarBudget+5*time.Second)
	}
	raw, err := os.ReadFile(started)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if m.sys.Run.Alive(pid) {
		t.Errorf("the standalone build %d still runs", pid)
	}
}
