package wire

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/background/adapter/launchd"
	"github.com/jterrazz/studio-station/apps/engine/background/adapter/smappservice"
	"github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	"github.com/jterrazz/studio-station/apps/engine/engine/process"
	"github.com/jterrazz/studio-station/apps/engine/platform/darwin"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func (m machine) jobs(budget time.Duration) contract.Jobs {
	if m.world != nil {
		return m.world.jobs()
	}
	if m.bundle() != "" {
		return m.bundledJobs(budget)
	}
	return m.bareJobs(budget)
}

func (m machine) bundledJobs(budget time.Duration) *smappservice.Jobs {
	return smappservice.New(m.bareJobs(budget), darwin.Agents{}, m.sys.Clock)
}

func (m machine) bareJobs(budget time.Duration) contract.Jobs {
	return launchd.New(m.sys.Run, m.sys.FS, m.sys.Clock, launchd.Options{
		Home: m.sys.Env.Home(), Root: m.systemRoot(), UID: m.sys.Env.Euid(), Budget: budget,
	})
}

// value: Spwn.app, when this binary is the engine it carries (ADR-073, mode
// A), its links resolved; "" for the bare binary of a Mac without the app,
// and in a world, whose jobs are the world's.
func (m machine) bundle() string {
	if m.world != nil {
		return ""
	}
	return local.BundleOf(m.sys.Env.Executable(), catalog.BundledEnginePath)
}

func (m machine) engineJobRef() contract.Ref {
	return contract.Ref{Domain: contract.Domain{UID: m.sys.Env.Euid()}, Label: engine.Label, File: catalog.EnginePlist(m.sys.Env.Home())}
}

func (m machine) agentJobRef() contract.Ref {
	return contract.Ref{Domain: contract.Domain{UID: m.sys.Env.Euid()}, Label: catalog.AgentLabel, File: catalog.AgentPlist(m.sys.Env.Home())}
}

type engineJob struct {
	jobs contract.Jobs
	ref  contract.Ref
}

func (m machine) engineJob() engineJob {
	return engineJob{jobs: m.jobs(process.JobBudget), ref: m.engineJobRef()}
}

func (j engineJob) Loaded(ctx context.Context) bool { return j.jobs.Read(ctx, j.ref).Loaded }

func (j engineJob) Load(ctx context.Context) error { return booted(j.jobs.Load(ctx, j.ref)) }

func (j engineJob) Start(ctx context.Context) error { return booted(j.jobs.Start(ctx, j.ref)) }

func (j engineJob) Stop(ctx context.Context) error { return booted(j.jobs.Stop(ctx, j.ref)) }

func booted(err error) error {
	var failed *contract.Failed
	if !errors.As(err, &failed) {
		return err
	}
	cause := failed.Err
	if cause == nil {
		cause = errors.New("exited " + strconv.Itoa(failed.Exit) + " " + failed.Stderr)
	}
	return &enginecalls.BootError{Message: failed.Verb() + ": " + cause.Error(), See: failed.See, Cause: cause}
}
