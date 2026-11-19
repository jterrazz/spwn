package fake

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/background"
	"github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

const command = "jobs"

type Jobs struct {
	w   *world.World
	mac string
	pid int
}

var _ contract.Jobs = (*Jobs)(nil)

func New(w *world.World, mac string) *Jobs { return &Jobs{w: w, mac: mac} }

func (j *Jobs) thisMac() *world.Mac {
	if j.w.Macs == nil {
		j.w.Macs = map[string]*world.Mac{}
	}
	m := j.w.Macs[j.mac]
	if m == nil {
		m = &world.Mac{}
		j.w.Macs[j.mac] = m
	}
	if m.Jobs == nil {
		m.Jobs = map[string]*world.Job{}
	}
	return m
}

func (j *Jobs) Present() bool {
	j.w.Lock()
	defer j.w.Unlock()
	return j.w.Macs[j.mac] != nil
}

func (j *Jobs) Names() contract.Names {
	return contract.Names{Manager: "the job manager", Command: command}
}

func (j *Jobs) failed(verb string, r contract.Ref) error {
	return &contract.Failed{Argv: []string{command, verb, r.Label}, Exit: 1, Stderr: r.Label + " is not loaded", See: command + " read " + r.Label}
}

func (j *Jobs) List(context.Context) ([]background.Job, error) {
	j.w.Lock()
	defer j.w.Unlock()
	m := j.thisMac()
	jobs := make([]background.Job, 0, len(m.Jobs)+1)
	for label, job := range m.Jobs {
		if label != catalog.EngineLabel {
			jobs = append(jobs, job.Job)
		}
	}
	if j.setUp() {
		jobs = append(jobs, j.engineJob())
	}
	slices.SortFunc(jobs, func(a, b background.Job) int { return strings.Compare(a.Label, b.Label) })
	return jobs, nil
}

// value: the pid the engine's job reads while it runs.
const enginePID = 812

// value: the engine's job as the job manager lists it: loaded and running
// while the engine runs, its definition where setup wrote it.
func (j *Jobs) engineJob() background.Job {
	m := j.thisMac()
	job := background.Job{Label: catalog.EngineLabel, Scope: background.ScopeUser, File: catalog.EnginePlist("~"), ProgramExists: true, OwnerExists: true}
	if written := m.Jobs[catalog.EngineLabel]; written != nil {
		job.File, job.Program = written.File, written.Program
	}
	if m.EngineRuns() {
		job.Loaded, job.Running, job.PID = true, true, enginePID
	}
	return job
}

// invariant: the engine's job is its Mac's engine: it runs while the world
// says the engine runs, a load sets it up, a start or a restart runs it, and
// a stop leaves it set up and stopped; a world that seeds an engine has its
// job without writing it.
func (j *Jobs) engine(r contract.Ref) bool { return r.Label == catalog.EngineLabel }

func (j *Jobs) setUp() bool {
	m := j.thisMac()
	return m.Engine != nil || m.Jobs[catalog.EngineLabel] != nil
}

func (j *Jobs) run(on bool) {
	m := j.thisMac()
	if m.Engine == nil {
		m.Engine = &world.Engine{Stopped: true}
	}
	if on && m.Engine.Stopped {
		m.Engine.StartedAt = j.w.Now
	}
	m.Engine.Stopped = !on
}

func (j *Jobs) Read(_ context.Context, r contract.Ref) contract.Status {
	j.w.Lock()
	defer j.w.Unlock()
	if j.engine(r) {
		if m := j.thisMac(); m.EngineRuns() {
			return contract.Status{Loaded: true, State: "running"}
		}
		return contract.Status{}
	}
	job := j.thisMac().Jobs[r.Label]
	switch {
	case job == nil || !job.Loaded:
		return contract.Status{}
	case job.Running:
		return contract.Status{Loaded: true, State: "running"}
	}
	return contract.Status{Loaded: true, State: "waiting"}
}

func (j *Jobs) Overrides(context.Context, contract.Domain) map[string]bool {
	j.w.Lock()
	defer j.w.Unlock()
	return maps.Clone(j.thisMac().Overrides)
}

func (j *Jobs) Install(_ context.Context, spec contract.JobSpec) (bool, error) {
	j.w.Lock()
	defer j.w.Unlock()
	if j.w.Fails("Jobs.Install") {
		return false, &contract.Failed{Argv: []string{command, "install", spec.Label}, Exit: 1, Stderr: "the job manager refused it", See: command + " read " + spec.Label}
	}
	definition := fmt.Sprintf("%+v", spec)
	m := j.thisMac()
	job := m.Jobs[spec.Label]
	if job != nil && job.Definition == definition {
		return false, nil
	}
	if job == nil {
		job = &world.Job{}
		m.Jobs[spec.Label] = job
	}
	job.Label, job.File, job.Scope, job.Definition = spec.Label, spec.File, background.ScopeUser, definition
	job.Program = ""
	if len(spec.Program) > 0 {
		job.Program = spec.Program[0]
	}
	job.ProgramExists, job.OwnerExists = true, true
	return true, nil
}

func (j *Jobs) Load(_ context.Context, r contract.Ref) error {
	j.w.Lock()
	defer j.w.Unlock()
	if j.engine(r) && j.setUp() {
		if j.thisMac().Engine == nil {
			j.run(false)
		}
		return nil
	}
	job := j.thisMac().Jobs[r.Label]
	if job == nil || job.File != r.File {
		return &contract.Failed{Argv: []string{command, "load", r.Label}, Exit: 1, Stderr: "no definition at " + r.File, See: command + " read " + r.Label}
	}
	job.Loaded = true
	return nil
}

func (j *Jobs) Start(_ context.Context, r contract.Ref) error {
	j.w.Lock()
	defer j.w.Unlock()
	if j.w.Fails("Jobs.Start") {
		return &contract.Failed{Argv: []string{command, "start", r.Label}, Exit: 5, Stderr: "the job manager could not start it", See: command + " read " + r.Label}
	}
	return j.start(r)
}

func (j *Jobs) start(r contract.Ref) error {
	if j.engine(r) {
		if !j.setUp() {
			return j.failed("start", r)
		}
		j.run(true)
		return nil
	}
	job := j.thisMac().Jobs[r.Label]
	if job == nil || !job.Loaded {
		return j.failed("start", r)
	}
	if !job.Running {
		j.pid++
		job.Running, job.PID, job.LastExit = true, j.pid, 0
	}
	return nil
}

func (j *Jobs) Restart(_ context.Context, r contract.Ref) error {
	j.w.Lock()
	defer j.w.Unlock()
	if j.engine(r) {
		return j.start(r)
	}
	job := j.thisMac().Jobs[r.Label]
	if job == nil || !job.Loaded {
		return j.failed("restart", r)
	}
	job.Running = false
	return j.start(r)
}

func (j *Jobs) Stop(_ context.Context, r contract.Ref) error {
	j.w.Lock()
	defer j.w.Unlock()
	return j.stop(r)
}

func (j *Jobs) stop(r contract.Ref) error {
	if j.engine(r) {
		if !j.thisMac().EngineRuns() {
			return j.failed("stop", r)
		}
		j.run(false)
		return nil
	}
	job := j.thisMac().Jobs[r.Label]
	if job == nil || !job.Loaded {
		return j.failed("stop", r)
	}
	job.Loaded, job.Running, job.PID = false, false, 0
	return nil
}

func (j *Jobs) Remove(_ context.Context, r contract.Ref) error {
	j.w.Lock()
	defer j.w.Unlock()
	_ = j.stop(r)
	delete(j.thisMac().Jobs, r.Label)
	return nil
}
