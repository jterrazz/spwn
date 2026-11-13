package process

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	awakecontract "github.com/jterrazz/studio-station/apps/engine/awake/contract"

	"github.com/jterrazz/studio-station/apps/engine/catalog"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"

	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
)

const JobBudget = startWait

const (
	bootoutWait = 10 * time.Second
	startWait   = 15 * time.Second
	controlPoll = 200 * time.Millisecond
)

type ControlMachines interface {
	Records() ([]model.VMRecord, error)
	StopDirect(ctx context.Context, name string) error
}

type Job interface {
	Loaded(ctx context.Context) bool
	Load(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type ControlOptions struct {
	// value: this Mac's power, whose clamshell a stop resets; nil is none.
	Power       awakecontract.Power
	ThisMacName string
	State       string
	// value: this station's version, which a reading compares with the engine's.
	Version string
	// value: where a stop the owner asks is kept, so a reading names it.
	Marks engine.StopMarks
}

type Control struct {
	// value: station engine stop and start: the one engine of this Mac, its
	// launchd agent and the VMs it supervises.
	h   Mac
	c   Peer
	m   ControlMachines
	hv  Hypervisor
	job Job
	o   ControlOptions
}

func NewControl(h Mac, c Peer, m ControlMachines, hv Hypervisor, job Job, o ControlOptions) *Control {
	return &Control{h: h, c: c, m: m, hv: hv, job: job, o: o}
}

func (c *Control) Check() error {
	if !c.installed() {
		return &enginecalls.NotInstalledError{}
	}
	return nil
}

func (c *Control) installed() bool {
	return c.h.Exists(catalog.EnginePlist(c.h.Home()))
}

func (c *Control) Read(ctx context.Context) (engine.Reading, error) {
	// invariant: a reading never fails for the engine's state: an engine that
	// does not answer is stopped (its agent kept) or not installed.
	r := engine.Reading{Station: c.o.Version}
	hello, err := c.c.Hello(ctx)
	if err != nil {
		r.State = engine.EngineNotInstalled
		if c.installed() {
			r.State = engine.EngineStopped
			if c.o.Marks != nil {
				r.Stopped, _ = c.o.Marks.Read()
			}
		}
		return r, nil
	}
	r.State, r.Version = engine.EngineRunning, hello.Version
	if !hello.StartedAt.IsZero() {
		r.Up = c.h.Now().Sub(hello.StartedAt)
	}
	if views, err := c.c.VMs(ctx); err == nil {
		for _, v := range views {
			if v.State == engine.StateRunning {
				r.VMs++
			}
		}
	}
	if v, err := c.c.AwakeView(ctx); err == nil && v.Session != nil {
		until := v.Session.Until
		r.AwakeUntil = &until
	}
	return r, nil
}

func (c *Control) Plan(ctx context.Context) (engine.Work, error) {
	// invariant: what a stop ends is every VM that runs, whatever its lifetime:
	// a stop keeps every disk, and only a delete destroys.
	records, err := c.m.Records()
	if err != nil {
		return engine.Work{}, err
	}
	var w engine.Work
	running := map[string]bool{}
	if views, err := c.c.VMs(ctx); err == nil {
		w.Answering = true
		for _, v := range views {
			running[v.Name] = v.State == engine.StateRunning
		}
	} else {
		for _, r := range records {
			_, running[r.Name] = c.hv.Adopt(ctx, r.Name)
		}
	}
	for _, r := range records {
		switch {
		case !running[r.Name]:
		case r.Lifetime == model.AlwaysOn:
			w.Servers = append(w.Servers, r.Name)
		default:
			w.VMs = append(w.VMs, engine.Stopped{Name: r.Name, Lifetime: string(r.Lifetime), Result: engine.ResultStopped})
		}
	}
	return w, nil
}

func (c *Control) Stop(ctx context.Context, w engine.Work, by string) (engine.StopReport, error) {
	r := engine.StopReport{Servers: w.Servers}
	if w.Answering {
		all, err := c.c.StopAll(ctx)
		if err != nil {
			return r, err
		}
		r.VMs, r.Sharing = all.VMs, all.SharingEnded
		if all.AwakeEnded {
			r.Awake = engine.AwakeEnded
		}
	} else {
		r.VMs = c.stopDirect(ctx, w)
		// invariant: a clamshell mask a dead engine left is released here, by the
		// same call the engine makes, so a stop never leaves the lid held.
		if c.o.Power != nil && c.o.Power.Clamshell(false) == nil {
			r.Awake = engine.AwakeReleased
		}
	}
	for _, v := range r.VMs {
		if v.Result == engine.ResultFailed {
			r.Engine = engine.EngineKept
			return r, &enginecalls.StuckError{VM: v.Name}
		}
	}
	if len(w.Servers) > 0 {
		r.Engine = engine.EngineServers
		return r, nil
	}
	if !w.Answering && !c.job.Loaded(ctx) {
		r.Engine = engine.EngineAlready
		return r, nil
	}
	if err := c.job.Stop(ctx); err != nil {
		return r, err
	}
	for waited := time.Duration(0); waited < bootoutWait && c.answers(ctx); waited += controlPoll {
		if err := c.h.Sleep(ctx, controlPoll); err != nil {
			return r, err
		}
	}
	r.Engine = engine.EngineHalted
	if c.o.Marks != nil {
		_ = c.o.Marks.Write(engine.StopMark{At: c.h.Now().UTC().Truncate(time.Second), By: by})
	}
	return r, nil
}

// invariant: with the engine down, a VM whose run outlived it is ended here,
// by the same path an engine-less stop takes, so nothing station started is
// left running once stop returns.
func (c *Control) stopDirect(ctx context.Context, w engine.Work) []engine.Stopped {
	out := make([]engine.Stopped, 0, len(w.VMs))
	for _, v := range w.VMs {
		if err := c.m.StopDirect(ctx, v.Name); err != nil {
			v.Result, v.Detail = engine.ResultFailed, err.Error()
		}
		out = append(out, v)
	}
	return out
}

func (c *Control) Start(ctx context.Context) (engine.StartReport, error) {
	if p, err := c.c.Hello(ctx); err == nil {
		return engine.StartReport{Already: true, Version: p.Version, PID: p.PID}, nil
	}
	if !c.job.Loaded(ctx) {
		if err := c.job.Load(ctx); err != nil {
			return engine.StartReport{}, c.failed(err)
		}
	}
	if err := c.job.Start(ctx); err != nil {
		return engine.StartReport{}, c.failed(err)
	}
	for waited := time.Duration(0); waited < startWait; waited += controlPoll {
		if p, err := c.c.Hello(ctx); err == nil {
			return engine.StartReport{Version: p.Version, PID: p.PID}, nil
		}
		if err := c.h.Sleep(ctx, controlPoll); err != nil {
			return engine.StartReport{}, err
		}
	}
	return engine.StartReport{}, c.failed(errors.New("it did not answer in " + strconv.Itoa(int(startWait.Seconds())) + "s"))
}

func (c *Control) failed(cause error) error {
	last := cause.Error()
	if data, err := c.h.ReadFile(engine.LogPath(c.o.State)); err == nil {
		lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		if line := strings.TrimSpace(lines[len(lines)-1]); line != "" {
			last = line
		}
	}
	return &engine.StartFailedError{Last: last, Cause: cause}
}

func (c *Control) answers(ctx context.Context) bool {
	_, err := c.c.Hello(ctx)
	return err == nil
}
