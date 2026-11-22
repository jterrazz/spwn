package wire

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

// value: a job manager that holds the engine's and the agent's jobs,
// rewrites a definition only when changed (or agentChanged, for the agent's
// alone) says so, and counts by label the loads, starts and restarts asked.
type installJobs struct {
	contract.Jobs
	changed, agentChanged, loaded bool
	agentPID                      int
	asked                         map[string]int
}

func (j *installJobs) count(verb string, r contract.Ref) error {
	if j.asked == nil {
		j.asked = map[string]int{}
	}
	j.asked[verb+" "+r.Label]++
	return nil
}

func (j *installJobs) Install(_ context.Context, spec contract.JobSpec) (bool, error) {
	return j.changed || (j.agentChanged && spec.Label == catalog.AgentLabel), nil
}
func (j *installJobs) Read(_ context.Context, r contract.Ref) contract.Status {
	if r.Label == catalog.AgentLabel && j.loaded && j.agentPID != 0 {
		return contract.Status{Loaded: true, State: "running", PID: j.agentPID}
	}
	return contract.Status{Loaded: j.loaded}
}
func (j *installJobs) Load(_ context.Context, r contract.Ref) error    { return j.count("load", r) }
func (j *installJobs) Start(_ context.Context, r contract.Ref) error   { return j.count("start", r) }
func (j *installJobs) Restart(_ context.Context, r contract.Ref) error { return j.count("restart", r) }

// value: the engine's socket; it answers a hello naming version, or refuses
// the dial when version is empty.
type helloSockets struct {
	local.Sockets
	version string
}

type duplex struct {
	io.Reader
	io.Writer
	closer func() error
}

func (d duplex) Close() error { return d.closer() }

func (s helloSockets) DialUnix(context.Context, string, time.Duration) (local.Conn, error) {
	if s.version == "" {
		return nil, errors.New("connection refused")
	}
	request, asked := io.Pipe()
	answered, answer := io.Pipe()
	go func() {
		defer func() { _ = answer.Close() }()
		if _, err := bufio.NewReader(request).ReadBytes('\n'); err != nil {
			return
		}
		out, _ := json.Marshal(engine.Hello{Version: s.version})
		frame, _ := json.Marshal(map[string]any{"v": 1, "id": "1", "out": json.RawMessage(out), "done": true})
		_, _ = answer.Write(append(frame, '\n'))
	}()
	return duplex{Reader: answered, Writer: asked, closer: asked.Close}, nil
}

func installerOf(running string) engineInstaller {
	sys := local.System{Env: localtest.Env{Exe: "/home/u/.local/bin/station", HomeDir: "/home/u", OS: "darwin"}, Sockets: helloSockets{version: running}}
	return engineInstaller{m: machine{sys: sys, paths: setup.Paths{State: "/home/u/.local/state/station"}}}
}

func TestInstallRestartsTheEngineOnlyWhenItsDefinitionChangedOrItRunsAnotherVersion(t *testing.T) {
	station := Version()
	older := station + "-older"
	for _, c := range []struct {
		name    string
		jobs    installJobs
		running string
		want    engine.Installed
	}{
		{"a newer binary, the same definition", installJobs{loaded: true}, older, engine.Installed{Restarted: true, From: older}},
		{"a moved binary, the same version", installJobs{loaded: true, changed: true}, station, engine.Installed{Restarted: true}},
		{"a moved binary, another version", installJobs{loaded: true, changed: true}, older, engine.Installed{Restarted: true, From: older}},
		{"the same binary, the same version", installJobs{loaded: true}, station, engine.Installed{}},
		{"an engine that does not answer, the same definition", installJobs{loaded: true}, "", engine.Installed{}},
		{"an engine that does not answer, a moved binary", installJobs{loaded: true, changed: true}, "", engine.Installed{Restarted: true}},
		{"a job not loaded yet", installJobs{changed: true}, "", engine.Installed{Written: true}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := installerOf(c.running).install(context.Background(), &c.jobs)
			if err != nil || got != c.want {
				t.Fatalf("install = %+v, %v; want %+v", got, err, c.want)
			}
			if wantRestarts := map[bool]int{true: 1, false: 0}[c.want.Restarted]; c.jobs.asked["restart "+catalog.EngineLabel] != wantRestarts {
				t.Errorf("the job was restarted %d times, want %d", c.jobs.asked["restart "+catalog.EngineLabel], wantRestarts)
			}
		})
	}
}

func TestTheAgentsJobIsStartedOnceAndRestartedWithItsPlistOrTheEngine(t *testing.T) {
	station := Version()
	for _, c := range []struct {
		name    string
		jobs    installJobs
		running string
		want    map[string]int
	}{
		{"a job not loaded yet", installJobs{changed: true}, "", map[string]int{"load": 1, "start": 1}},
		{"the same plist, the same engine", installJobs{loaded: true}, station, map[string]int{}},
		{"a moved plist, the same engine", installJobs{loaded: true, agentChanged: true}, station, map[string]int{"restart": 1}},
		{"the same plist, an engine of another version", installJobs{loaded: true}, station + "-older", map[string]int{"restart": 1}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := installerOf(c.running).install(context.Background(), &c.jobs); err != nil {
				t.Fatal(err)
			}
			for _, verb := range []string{"load", "start", "restart"} {
				if got := c.jobs.asked[verb+" "+catalog.AgentLabel]; got != c.want[verb] {
					t.Errorf("the agent's job: %s asked %d times, want %d", verb, got, c.want[verb])
				}
			}
		})
	}
}

// value: the text files of one process, or no answer for it when files is
// nil.
type processTexts struct {
	pid   int
	files []string
}

func (p processTexts) Texts(_ context.Context, pid int) ([]string, error) {
	if pid != p.pid || p.files == nil {
		return nil, errors.New("no such process")
	}
	return p.files, nil
}

func TestTheAgentsJobRestartsWhenItsProcessRunsAnotherFileThanTheOneItStarts(t *testing.T) {
	const pid = 9006
	for _, c := range []struct {
		name     string
		files    []string
		restarts int
	}{
		{"the file its job starts", []string{"/home/u/.local/bin/station", "/usr/lib/dyld"}, 0},
		{"a file moved aside by an update", []string{"/private/tmp/old-Spwn.app/Contents/MacOS/Spwn Engine", "/usr/lib/dyld"}, 1},
		{"a process whose files are not read", nil, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			installer := installerOf(Version())
			installer.texts = processTexts{pid: pid, files: c.files}
			jobs := installJobs{loaded: true, agentPID: pid}
			if _, err := installer.install(context.Background(), &jobs); err != nil {
				t.Fatal(err)
			}
			if got := jobs.asked["restart "+catalog.AgentLabel]; got != c.restarts {
				t.Errorf("the agent's job was restarted %d times, want %d", got, c.restarts)
			}
			if got := jobs.asked["restart "+catalog.EngineLabel]; got != 0 {
				t.Errorf("the engine's job was restarted %d times, want 0", got)
			}
		})
	}
}

func TestTheAgentsJobRunsTheAgentAtLoginAndKeepsItAlive(t *testing.T) {
	jobs := &specJobs{}
	if _, err := installerOf("").install(context.Background(), jobs); err != nil {
		t.Fatal(err)
	}
	spec, ok := jobs.specs[catalog.AgentLabel]
	if !ok {
		t.Fatal("no job was written for the agent")
	}
	if got := strings.Join(spec.Program, " "); got != "/home/u/.local/bin/station machines agent" {
		t.Errorf("program = %q", got)
	}
	if !spec.AtLogin || !spec.KeepAlive || spec.File != catalog.AgentPlist("/home/u") || spec.Log != "/home/u/.local/state/station/logs/agent.log" {
		t.Errorf("spec = %+v", spec)
	}
}

// value: a job manager that keeps every definition it is given.
type specJobs struct {
	installJobs
	specs map[string]contract.JobSpec
}

func (j *specJobs) Install(_ context.Context, spec contract.JobSpec) (bool, error) {
	if j.specs == nil {
		j.specs = map[string]contract.JobSpec{}
	}
	j.specs[spec.Label] = spec
	return true, nil
}
