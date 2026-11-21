package smappservice

import (
	"context"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/background/contract"
)

type Agents interface {
	// value: SMAppService's agents of the app whose bundle holds this binary,
	// each named by its plist in Contents/Library/LaunchAgents.
	Register(plist string) error
	Unregister(plist string) error
	// value: the app holds the agent registered and enabled.
	Registered(plist string) bool
}

type Clock interface {
	Sleep(ctx context.Context, d time.Duration) error
}

type Jobs struct {
	// value: the jobs of the engine inside Spwn.app (ADR-073, mode A): their
	// definitions are the bundle's, registered by SMAppService, and every read,
	// start and stop is launchd's, as on a Mac without the app; whether a job
	// runs is judged by `launchctl print`, never by SMAppService's status,
	// which reads enabled for a job launchd does not hold, and that status
	// answers only whether the app holds the job installed.
	contract.Jobs
	agents Agents
	clock  Clock
}

var _ contract.Jobs = (*Jobs)(nil)

func New(manager contract.Jobs, agents Agents, clock Clock) *Jobs {
	return &Jobs{Jobs: manager, agents: agents, clock: clock}
}

func plistOf(r contract.Ref) string { return r.Label + ".plist" }

// value: the read that judges a job, its job manager's own.
func (j *Jobs) read(r contract.Ref) string {
	return j.Names().Command + " print gui/$(id -u)/" + r.Label
}

func (j *Jobs) failed(verb string, r contract.Ref, err error) error {
	return &contract.Failed{
		Argv: []string{"SMAppService", verb, plistOf(r)}, Exit: 1, Stderr: err.Error(),
		See: j.read(r),
	}
}

func (j *Jobs) Installed(r contract.Ref) bool {
	// invariant: a bundled job is installed while its app holds it
	// registered, read from SMAppService, whatever ~/Library/LaunchAgents
	// holds.
	return j.agents.Registered(plistOf(r))
}

func (j *Jobs) Install(context.Context, contract.JobSpec) (bool, error) {
	// invariant: a sealed bundle's definition is never written: nothing changed.
	return false, nil
}

func (j *Jobs) Load(_ context.Context, r contract.Ref) error {
	if err := j.agents.Register(plistOf(r)); err != nil {
		return j.failed("register", r, err)
	}
	return nil
}

func (j *Jobs) Restart(ctx context.Context, r contract.Ref) error {
	// invariant: a restart unregisters then registers the job, so launchd runs
	// the binary the bundle holds now: `kickstart -k` is never trusted across a
	// replaced binary (ADR-073).
	_ = j.agents.Unregister(plistOf(r))
	if _, err := j.unloaded(ctx, r, unloadPolls); err != nil {
		return err
	}
	if err := j.Load(ctx, r); err != nil {
		return err
	}
	return j.Start(ctx, r)
}

func (j *Jobs) Retire(ctx context.Context, r contract.Ref) error {
	// invariant: a bare job of the same label leaves launchd before the bundle
	// registers it, and is never unregistered: SMAppService's unregister of a
	// label a bare job still holds disables that job's Background Task
	// Management record, and the register that follows is refused, "Job is not
	// allowed to bootstrap" (ADR-073). Its bootout returns while its process
	// still exits, so the label is waited out of launchd, up to launchd's own
	// exit timeout.
	if err := j.Jobs.Remove(ctx, r); err != nil {
		return err
	}
	gone, err := j.unloaded(ctx, r, retirePolls)
	if err != nil || gone {
		return err
	}
	return &contract.Failed{
		Argv: []string{j.Names().Command, "bootout", "gui/" + strconv.Itoa(r.UID) + "/" + r.Label}, Exit: 1,
		Stderr: j.Names().Manager + " still holds the job " + (retirePolls * unloadPoll).String() + " after its bootout",
		See:    j.read(r),
	}
}

func (j *Jobs) unloaded(ctx context.Context, r contract.Ref, polls int) (bool, error) {
	for wait := 0; j.Read(ctx, r).Loaded; wait++ {
		if wait == polls {
			return false, nil
		}
		if err := j.clock.Sleep(ctx, unloadPoll); err != nil {
			return false, err
		}
	}
	return true, nil
}

// value: how long a restart waits for launchd to drop the job it
// unregistered, and a retire for the bare job it booted out: launchd's
// default ExitTimeOut, 20 s, before it kills the process.
const (
	unloadPoll  = 100 * time.Millisecond
	unloadPolls = 50
	retirePolls = 200
)

func (j *Jobs) Remove(ctx context.Context, r contract.Ref) error {
	// invariant: a job no longer registered is no failure, as a definition
	// already gone is none.
	_ = j.Stop(ctx, r)
	_ = j.agents.Unregister(plistOf(r))
	return nil
}
