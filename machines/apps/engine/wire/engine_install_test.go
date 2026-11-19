package wire

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

// value: a job manager that holds the engine's job, rewrites its definition
// only when changed says so, and counts the restarts it is asked for.
type installJobs struct {
	contract.Jobs
	changed, loaded bool
	restarts, loads int
}

func (j *installJobs) Install(context.Context, contract.JobSpec) (bool, error) { return j.changed, nil }
func (j *installJobs) Read(context.Context, contract.Ref) contract.Status {
	return contract.Status{Loaded: j.loaded}
}
func (j *installJobs) Load(context.Context, contract.Ref) error    { j.loads++; return nil }
func (j *installJobs) Start(context.Context, contract.Ref) error   { return nil }
func (j *installJobs) Restart(context.Context, contract.Ref) error { j.restarts++; return nil }

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
			if wantRestarts := map[bool]int{true: 1, false: 0}[c.want.Restarted]; c.jobs.restarts != wantRestarts {
				t.Errorf("the job was restarted %d times, want %d", c.jobs.restarts, wantRestarts)
			}
		})
	}
}
