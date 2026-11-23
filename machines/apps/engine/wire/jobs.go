package wire

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/background/adapter/launchd"
	"github.com/jterrazz/studio-station/apps/engine/background/adapter/smappservice"
	"github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
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
		Bundle: m.bundle(), Executable: local.ExecutableOf,
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

// value: this Mac's one engine job: the bundle's own label and definition
// in mode A, the bare binary's in mode B.
func (m machine) engineJobRef() contract.Ref {
	domain := contract.Domain{UID: m.sys.Env.Euid()}
	if bundle := m.bundle(); bundle != "" {
		return contract.Ref{Domain: domain, Label: catalog.BundledEngineLabel, File: filepath.Join(bundle, catalog.BundledPlist(catalog.BundledEngineLabel))}
	}
	return contract.Ref{Domain: domain, Label: catalog.EngineLabel, File: catalog.EnginePlist(m.sys.Env.Home())}
}

// value: the bare binary's jobs, the key agent's an earlier release wrote
// first, as the move to the bundle boots them out.
func (m machine) bareJobRefs() []contract.Ref {
	domain := contract.Domain{UID: m.sys.Env.Euid()}
	home := m.sys.Env.Home()
	return []contract.Ref{
		{Domain: domain, Label: catalog.RetiredAgentLabel, File: catalog.RetiredAgentPlist(home)},
		{Domain: domain, Label: catalog.EngineLabel, File: catalog.EnginePlist(home)},
	}
}

type engineJob struct {
	jobs contract.Jobs
	ref  contract.Ref
}

func (m machine) engineJob() engineJob {
	return engineJob{jobs: m.jobs(process.JobBudget), ref: m.engineJobRef()}
}

func (j engineJob) Loaded(ctx context.Context) bool { return j.jobs.Read(ctx, j.ref).Loaded }

func (j engineJob) Installed() bool { return j.jobs.Installed(j.ref) }

// value: this Mac's engine job is installed, as its job manager answers,
// in either mode; every reader of "installed" asks it here.
func (m machine) engineInstalled() bool { return m.engineJob().Installed() }

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
