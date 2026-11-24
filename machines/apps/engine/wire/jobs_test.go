package wire

import (
	"context"
	"errors"
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
	"github.com/jterrazz/studio-station/apps/engine/setup"
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

func TestTheBundlesJobIsTheAppsOwnLabelSoNoRecordOfTheBareJobsMatchesIt(t *testing.T) {
	m := machineRunning(t, true)
	want := contract.Ref{Domain: contract.Domain{UID: 501}, Label: "com.jterrazz.spwn.engine", File: m.bundle() + "/Contents/Library/LaunchAgents/com.jterrazz.spwn.engine.plist"}
	if got := m.engineJobRef(); got != want {
		t.Errorf("the bundle's job = %+v, want %+v", got, want)
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

func TestTheBareBinarysJobIsTheBareEngineTheMoveRetires(t *testing.T) {
	m := machineRunning(t, false)
	if got := m.engineJobRef(); got != m.bareJobRefs()[1] {
		t.Errorf("the bare binary's job = %+v, want %+v", got, m.bareJobRefs()[1])
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
		if err := (engineInstaller{m: m}).retireStandaloneMenuBar(context.Background()); err != nil {
			t.Fatal(err)
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
	if err := (engineInstaller{m: m}).retireStandaloneMenuBar(ctx); err != nil {
		t.Fatal(err)
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

type loginItems struct {
	shown []string
	err   error
}

func (l *loginItems) ShowLoginItem(identifier string) error {
	l.shown = append(l.shown, identifier)
	return l.err
}

func (l *loginItems) HideLoginItem(string) error { return l.err }

// value: a fresh Mac that runs the engine Spwn.app carries, over the real
// files of a home of its own, and the menu bar's login item it asks macOS for.
func freshBundledMac(t *testing.T, items *loginItems) (installer engineInstaller, home string) {
	t.Helper()
	home = t.TempDir()
	m := machineRunning(t, true)
	exe := m.sys.Env.Executable()
	m.sys = local.New(nil, nil, io.Discard, io.Discard)
	m.sys.Env = localtest.Env{Exe: exe, HomeDir: home, OS: "darwin", UID: 501}
	return engineInstaller{m: m, menuBar: items}, home
}

func TestTheFirstMoveIntoTheBundleShowsTheMenuBarOnAFreshMac(t *testing.T) {
	for _, refused := range []error{nil, errors.New("SMAppServiceErrorDomain 1: Operation not permitted")} {
		items := &loginItems{err: refused}
		i, home := freshBundledMac(t, items)
		if _, err := i.intoTheBundle(context.Background(), i.m.bundle()); err != nil {
			t.Fatal(err)
		}
		if len(items.shown) != 1 || items.shown[0] != "com.jterrazz.station.menubar" {
			t.Errorf("refused=%v: the move asked macOS to show %v, want the menu bar once", refused, items.shown)
		}
		if _, err := i.m.sys.FS.Stat(filepath.Join(home, ".local", "state", "station", "menubar-offered")); err != nil {
			t.Errorf("refused=%v: the move kept no record that it showed the menu bar: %v", refused, err)
		}
	}
}

func TestAnUpdateAfterTheMoveNeverShowsTheMenuBarAgainSoThePersonsOffIsKept(t *testing.T) {
	items := &loginItems{}
	i, _ := freshBundledMac(t, items)
	for range 3 {
		if _, err := i.intoTheBundle(context.Background(), i.m.bundle()); err != nil {
			t.Fatal(err)
		}
	}
	if len(items.shown) != 1 {
		t.Errorf("three starts of the bundle asked macOS to show the menu bar %d times, want once", len(items.shown))
	}
}

// value: launchd's menu bar job, which macOS drops once its login item is
// unregistered and loads, running, once it is registered again.
type menuBarJobs struct {
	installJobs
	menuBar contract.Status
	asked   []string
}

func (j *menuBarJobs) Read(ctx context.Context, r contract.Ref) contract.Status {
	if r.Label == "com.jterrazz.station.menubar" {
		return j.menuBar
	}
	return j.installJobs.Read(ctx, r)
}

func (j *menuBarJobs) ShowLoginItem(identifier string) error {
	j.asked = append(j.asked, "show "+identifier)
	j.menuBar = contract.Status{Loaded: true, State: "running", PID: 7002}
	return nil
}

func (j *menuBarJobs) HideLoginItem(identifier string) error {
	j.asked = append(j.asked, "hide "+identifier)
	j.menuBar = contract.Status{}
	return nil
}

func TestAnUpdateRestartsTheMenuBarThatRanSoItRunsTheBundlesBinaryAndLeavesOneThatDidNot(t *testing.T) {
	station := Version()
	restarted := []string{"hide com.jterrazz.station.menubar", "show com.jterrazz.station.menubar"}
	for _, c := range []struct {
		name    string
		running string
		menuBar contract.Status
		want    []string
	}{
		{"the engine moves to the new version, the menu bar runs", station + "-older", contract.Status{Loaded: true, State: "running", PID: 7001}, restarted},
		{"the engine moves to the new version, the menu bar was quit", station + "-older", contract.Status{Loaded: true, State: "not running"}, nil},
		{"the engine moves to the new version, the menu bar is off", station + "-older", contract.Status{}, nil},
		{"the engine already runs this version, the menu bar runs", station, contract.Status{Loaded: true, State: "running", PID: 7001}, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			jobs := &menuBarJobs{installJobs: installJobs{loaded: true}, menuBar: c.menuBar}
			i, home := freshBundledMac(t, &loginItems{})
			i.menuBar = jobs
			i.m.sys.Sockets = helloSockets{version: c.running}
			i.m.paths = setup.Paths{State: filepath.Join(home, ".local", "state", "station")}
			if err := os.MkdirAll(i.m.paths.State, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(i.m.paths.State, "menubar-offered"), nil, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := i.install(context.Background(), jobs); err != nil {
				t.Fatal(err)
			}
			if strings.Join(jobs.asked, ", ") != strings.Join(c.want, ", ") {
				t.Errorf("the update asked macOS %v, want %v", jobs.asked, c.want)
			}
		})
	}
}
