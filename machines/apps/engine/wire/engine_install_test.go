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
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

// value: a job manager that holds the engine's job, rewrites its
// definition only when changed says so, and counts by label the loads,
// starts and restarts asked.
type installJobs struct {
	contract.Jobs
	changed, loaded bool
	asked           map[string]int
}

func (j *installJobs) count(verb string, r contract.Ref) error {
	if j.asked == nil {
		j.asked = map[string]int{}
	}
	j.asked[verb+" "+r.Label]++
	return nil
}

func (j *installJobs) Install(context.Context, contract.JobSpec) (bool, error) {
	return j.changed, nil
}
func (j *installJobs) Read(context.Context, contract.Ref) contract.Status {
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
	agent   model.Agent
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
		out, _ := json.Marshal(engine.Hello{Version: s.version, Agent: s.agent})
		frame, _ := json.Marshal(map[string]any{"v": 1, "id": "1", "out": json.RawMessage(out), "done": true})
		_, _ = answer.Write(append(frame, '\n'))
	}()
	return duplex{Reader: answered, Writer: asked, closer: asked.Close}, nil
}

func installerOf(running string, agent ...model.Agent) engineInstaller {
	hello := helloSockets{version: running}
	if len(agent) > 0 {
		hello.agent = agent[0]
	}
	sys := local.System{Env: localtest.Env{Exe: "/home/u/.local/bin/station", HomeDir: "/home/u", OS: "darwin"}, Sockets: hello}
	return engineInstaller{m: machine{sys: sys, paths: setup.Paths{State: "/home/u/.local/state/station"}}}
}

func TestInstallRestartsTheEngineOnlyWhenItsDefinitionChangedItRunsAnotherVersionOrServesNoAgent(t *testing.T) {
	station := Version()
	older := station + "-older"
	held := model.Agent{State: model.AgentHeld, PID: 9006, Free: "kill 9006"}
	for _, c := range []struct {
		name    string
		jobs    installJobs
		running string
		agent   model.Agent
		restart bool
		want    engine.Installed
	}{
		{"the same binary, the same version, its agent socket held", installJobs{loaded: true}, station, held, false, engine.Installed{Restarted: true}},
		{"the same binary, the same version, its agent failed", installJobs{loaded: true}, station, model.Agent{State: model.AgentFailed, Detail: "listen"}, false, engine.Installed{Restarted: true}},
		{"the same binary, the same version, its agent served", installJobs{loaded: true}, station, model.Agent{State: model.AgentServed}, false, engine.Installed{}},
		{"a newer binary, the same definition", installJobs{loaded: true}, older, model.Agent{}, false, engine.Installed{Restarted: true, From: older}},
		{"a moved binary, the same version", installJobs{loaded: true, changed: true}, station, model.Agent{}, false, engine.Installed{Restarted: true}},
		{"a moved binary, another version", installJobs{loaded: true, changed: true}, older, model.Agent{}, false, engine.Installed{Restarted: true, From: older}},
		{"the same binary, the same version", installJobs{loaded: true}, station, model.Agent{}, false, engine.Installed{}},
		{"an engine that does not answer, the same definition", installJobs{loaded: true}, "", model.Agent{}, false, engine.Installed{}},
		{"an engine that does not answer, a moved binary", installJobs{loaded: true, changed: true}, "", model.Agent{}, false, engine.Installed{Restarted: true}},
		{"a job not loaded yet", installJobs{changed: true}, "", model.Agent{}, false, engine.Installed{Written: true}},
		{"the same binary, the same version, a restart asked", installJobs{loaded: true}, station, model.Agent{State: model.AgentServed}, true, engine.Installed{Restarted: true}},
		{"a restart asked of an engine that does not answer", installJobs{loaded: true}, "", model.Agent{}, true, engine.Installed{Restarted: true}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := installerOf(c.running, c.agent).install(context.Background(), &c.jobs, c.restart)
			if err != nil || got != c.want {
				t.Fatalf("install = %+v, %v; want %+v", got, err, c.want)
			}
			if wantRestarts := map[bool]int{true: 1, false: 0}[c.want.Restarted]; c.jobs.asked["restart "+catalog.EngineLabel] != wantRestarts {
				t.Errorf("the job was restarted %d times, want %d", c.jobs.asked["restart "+catalog.EngineLabel], wantRestarts)
			}
		})
	}
}

func TestInstallWritesOneJobTheEnginesWhichRunsAtLoginAndIsKeptAlive(t *testing.T) {
	jobs := &specJobs{}
	if _, err := installerOf("").install(context.Background(), jobs, false); err != nil {
		t.Fatal(err)
	}
	if len(jobs.specs) != 1 {
		t.Fatalf("install wrote %d jobs, want the engine's alone: %+v", len(jobs.specs), jobs.specs)
	}
	spec, ok := jobs.specs[catalog.EngineLabel]
	if !ok {
		t.Fatalf("no job was written for the engine: %+v", jobs.specs)
	}
	if got := strings.Join(spec.Program, " "); got != "/home/u/.local/bin/station engine run" {
		t.Errorf("program = %q", got)
	}
	if !spec.AtLogin || !spec.KeepAlive || spec.File != catalog.EnginePlist("/home/u") || spec.Log != "/home/u/.local/state/station/logs/engine.log" {
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
