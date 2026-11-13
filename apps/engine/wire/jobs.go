package wire

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/background/adapter/launchd"
	"github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	"github.com/jterrazz/studio-station/apps/engine/engine/process"
)

func (m machine) jobs(budget time.Duration) contract.Jobs {
	if m.world != nil {
		return m.world.jobs()
	}
	return launchd.New(m.sys.Run, m.sys.FS, launchd.Options{
		Home: m.sys.Env.Home(), Root: m.systemRoot(), UID: m.sys.Env.Euid(), Budget: budget,
	})
}

func (m machine) engineJobRef() contract.Ref {
	return contract.Ref{Domain: contract.Domain{UID: m.sys.Env.Euid()}, Label: engine.Label, File: catalog.EnginePlist(m.sys.Env.Home())}
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
